package main

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"log"
	"net"
	"net/http"

	db "github.com/MumAroi/go-simplebank/db/sqlc"
	"github.com/MumAroi/go-simplebank/gapi"
	"github.com/MumAroi/go-simplebank/pb"
	"github.com/MumAroi/go-simplebank/util"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:embed doc/swagger
var webFiles embed.FS

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("can not load config:", err)
	}

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("can not connect to db:", err)
	}

	runDBMigration(config.MigrationURL, config.DBSource)

	store := db.NewStore(conn)
	go runGatewayServer(config, store)
	runGrpcServer(config, store)
}

func runDBMigration(migrationURL string, dbSource string) {
	migration, err := migrate.New(migrationURL, dbSource)
	if err != nil {
		log.Fatal("can not create migration:", err)
	}

	if err := migration.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("can not run migration:", err)
	}

	log.Println("db migrate successfully")
}

func runGrpcServer(config util.Config, store db.Store) {
	server, err := gapi.NewServer(config, store)
	if err != nil {
		log.Fatal("can not create server:", err)
	}

	// สร้าง gRPC runtime server ซึ่งทำหน้าที่รับ connection และจัดการ RPC requests
	grpcServer := grpc.NewServer()

	// ลงทะเบียน SimpleBank service โดยให้ gRPC ส่งแต่ละ RPC ไปยัง methods ของ gapi.Server
	pb.RegisterSimpleBankServer(grpcServer, server)

	// เปิด Server Reflection เพื่อให้เครื่องมือ เช่น Evans หรือ Bruno อ่าน schema ได้โดยไม่ต้องรับไฟล์ .proto
	// Server Reflection คือความสามารถที่เปิดให้ gRPC Client สอบถาม Server ได้ว่า Server มี API อะไรบ้าง โดยไม่ต้องมีไฟล์ `.proto` อยู่ฝั่ง Client
	reflection.Register(grpcServer)

	// เปิด TCP listener ตาม address ที่กำหนด เช่น 0.0.0.0:3011
	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		log.Fatal("can not listen on grpc server:", err)
	}

	log.Printf("start gRPC server at %s", listener.Addr().String())

	// เริ่มรับ requests จาก listener; Serve จะ block อยู่จนกว่า Server จะหยุดหรือเกิด error
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatal("can not serve grpc server:", err)
	}
}

func runGatewayServer(config util.Config, store db.Store) {
	// สร้าง implementation ของ SimpleBank service ที่มี business logic, Store และ TokenMaker
	server, err := gapi.NewServer(config, store)
	if err != nil {
		log.Fatal("can not create server:", err)
	}

	// กำหนดวิธีแปลงระหว่าง HTTP JSON กับ Protobuf messages
	// UseProtoNames ทำให้ JSON ใช้ชื่อ field แบบ snake_case ตามไฟล์ .proto
	// DiscardUnknown ยอมข้าม JSON fields ที่ไม่มีอยู่ใน Protobuf message
	jsonOption := runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames: true,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	})

	// สร้าง gRPC-Gateway multiplexer ซึ่งรับ HTTP routes ที่ generate จาก annotations ใน .proto
	grpcMux := runtime.NewServeMux(jsonOption)

	// สร้าง Context สำหรับควบคุมอายุการทำงานของ Gateway handlers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ลงทะเบียน generated HTTP handlers และเรียก gapi.Server โดยตรงภายใน process เดียวกัน
	// รูปแบบ HandlerServer นี้ไม่ต้องเปิด gRPC client connection ไปยัง port ของ gRPC Server
	err = pb.RegisterSimpleBankHandlerServer(ctx, grpcMux, server)
	if err != nil {
		log.Fatal("can not register grpc server:", err)
	}

	// สร้าง HTTP multiplexer หลักเพื่อเลือก Handler จาก URL path ที่ request เข้ามา
	// Multiplexer หรือเรียกสั้น ๆ ว่า Mux คือตัวคัดแยก Request แล้วส่งไปยัง Handler ที่ตรงกับ Request นั้นครับ
	mux := http.NewServeMux()

	// ใช้ gRPC-Gateway เป็น Handler หลักสำหรับ HTTP API routes เช่น /v1/create_user
	mux.Handle("/", grpcMux)

	staticFiles, err := fs.Sub(webFiles, "doc/swagger")
	if err != nil {
		log.Fatal("can not create static files:", err)
	}

	// สร้าง FileServer สำหรับอ่าน static files จาก ./doc/swagger เช่น index.html, CSS และ JavaScript
	// fs := http.FileServer(http.Dir("./doc/swagger"))

	// ให้ requests ที่ขึ้นต้นด้วย /swagger/ ไปยัง FileServer และตัด prefix ก่อนค้นหาไฟล์
	// ตัวอย่าง /swagger/index.html จะถูกแปลงเป็น /index.html แล้วอ่าน ./doc/swagger/index.html
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.FS(staticFiles))))

	// เปิด TCP listener สำหรับ HTTP/JSON Gateway ตาม address เช่น 0.0.0.0:8080
	listener, err := net.Listen("tcp", config.HTTPServerAddress)
	if err != nil {
		log.Fatal("can not create listener:", err)
	}

	log.Printf("start http gateway server at %s", listener.Addr().String())

	// เริ่มรับ HTTP requests; Serve จะ block จนกว่า Server จะหยุดหรือเกิด error
	err = http.Serve(listener, mux)
	if err != nil {
		log.Fatal("can not serve http gateway server:", err)
	}
}

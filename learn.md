# Go Simple Bank — Concise Learning Notes

เอกสารนี้สรุปแนวคิดสำคัญของโปรเจกต์ SimpleBank โดยเรียงจากฐานข้อมูลไปถึง API, tests และ CI

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Database Migrations and PostgreSQL](#2-database-migrations-and-postgresql)
3. [Transactions, ACID and Deadlocks](#3-transactions-acid-and-deadlocks)
4. [sqlc](#4-sqlc)
5. [Store, Interfaces and Dependency Injection](#5-store-interfaces-and-dependency-injection)
6. [Common Go Basics](#6-common-go-basics)
7. [Context](#7-context)
8. [Testing](#8-testing)
9. [GoMock and HTTP Handler Testing](#9-gomock-and-http-handler-testing)
10. [Coverage](#10-coverage)
11. [GitHub Actions and CI](#11-github-actions-and-ci)
12. [Common Commands](#12-common-commands)

## 1. System Overview

โครงสร้างหลักของโปรเจกต์:

```text
HTTP Client
    ↓
Gin Handler
    ↓
Store interface
    ↓
SQLStore
    ↓
sqlc Queries
    ↓
PostgreSQL
```

- Gin รับ request, validate input และสร้าง response
- `Store` กำหนดความสามารถที่ API ใช้
- `SQLStore` ทำงานกับ PostgreSQL จริง
- sqlc สร้าง Go methods จาก SQL
- PostgreSQL เก็บข้อมูลและควบคุม transaction

ตอน unit test เปลี่ยน `SQLStore` เป็น `MockStore` ได้:

```text
Gin Handler → MockStore → ค่าที่ test กำหนด
```

## 2. Database Migrations and PostgreSQL

Migration ใช้บันทึกการเปลี่ยนแปลง schema เป็นลำดับ:

```text
000001_init_schema.up.sql   → ใช้เพิ่มหรือแก้ schema
000001_init_schema.down.sql → ใช้ย้อนการเปลี่ยนแปลง
```

สร้าง migration:

```bash
migrate create -ext sql -dir db/migration -seq init_schema
```

ตัวอย่าง:

```sql
-- up
CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    owner VARCHAR NOT NULL,
    balance BIGINT NOT NULL CHECK (balance >= 0)
);

-- down
DROP TABLE accounts;
```

รันและย้อน migration:

```bash
make migrateup
make migratedown
```

หลักสำคัญ:

- `up` และ `down` ควรเป็นคู่กัน
- schema ที่ทุก environment ใช้ควรมาจาก migration
- Constraints เช่น PK, FK และ `CHECK` ควรบังคับในฐานข้อมูลเมื่อทำได้

### `UNIQUE CONSTRAINT` vs `UNIQUE INDEX`

- `UNIQUE CONSTRAINT` คือกฎของตารางว่าข้อมูลห้ามซ้ำ และ PostgreSQL จะสร้าง Unique Index ให้เอง
- `UNIQUE INDEX` คือสร้าง Index โดยตรง เหมาะกับเงื่อนไขพิเศษ เช่น `WHERE deleted_at IS NULL` หรือ `LOWER(email)`
- กรณีทั่วไป เช่น `(owner, currency)` ห้ามซ้ำ ควรใช้ Constraint เพียงอย่างเดียว

```sql
ALTER TABLE accounts
ADD CONSTRAINT owner_currency_key UNIQUE (owner, currency);
```

ไม่ต้องสร้าง Unique Index บน columns ชุดเดิมเพิ่ม เพราะจะซ้ำซ้อนและเพิ่มภาระตอนเขียนข้อมูล

### Constraints in Brief

Constraint คือกฎที่ PostgreSQL ใช้ตรวจความถูกต้องตอน `INSERT` และ `UPDATE` ถ้าข้อมูลผิดกฎจะไม่บันทึก

- `PRIMARY KEY` — รหัสหลัก ต้องไม่ซ้ำและไม่เป็น `NULL`
- `UNIQUE` — ค่าหรือชุดค่าห้ามซ้ำ
- `FOREIGN KEY` — ค่าที่อ้างอิงต้องมีอยู่ในอีกตาราง
- `CHECK` — ค่าต้องผ่านเงื่อนไขที่กำหนด
- `NOT NULL` — ห้ามไม่มีค่า

การลบ Constraint คือการลบเฉพาะกฎ ไม่ได้ลบตาราง Column หรือข้อมูล:

```sql
ALTER TABLE accounts
DROP CONSTRAINT IF EXISTS accounts_owner_fkey;
```

หลังลบ Foreign Key แล้ว PostgreSQL จะไม่ตรวจว่า `accounts.owner` มีอยู่ใน `users.username` อีกต่อไป

## 3. Transactions, ACID and Deadlocks

Transaction รวมหลาย SQL operations ให้เป็นงานเดียว:

```text
สำเร็จทั้งหมด → COMMIT
ผิดพลาดบางส่วน → ROLLBACK ทั้งหมด
```

### ACID

- Atomicity — สำเร็จทั้งหมดหรือไม่เกิดการเปลี่ยนแปลงเลย
- Consistency — ข้อมูลยังถูกต้องตาม constraints และ business rules
- Isolation — transactions ที่ทำพร้อมกันไม่รบกวนกันอย่างผิดพลาด
- Durability — commit แล้วข้อมูลต้องคงอยู่

### Isolation Levels

| ระดับ | พฤติกรรมโดยย่อ | ข้อควรระวัง |
| --- | --- | --- |
| Read Uncommitted | PostgreSQL ทำงานเหมือน Read Committed | ไม่ได้อ่านข้อมูลที่ยังไม่ commit |
| Read Committed | snapshot ใหม่ทุก statement | อ่านซ้ำอาจเห็นค่าใหม่ |
| Repeatable Read | snapshot เดิมตลอด transaction | อาจ conflict และต้อง retry |
| Serializable | ผลเสมือนทำ transactions ทีละตัว | อาจเกิด `40001` และต้อง retry |

PostgreSQL ใช้ `Read Committed` เป็นค่าเริ่มต้น

ปัญหาที่พบบ่อย:

- Dirty Read — อ่านข้อมูลที่ยังไม่ commit
- Non-repeatable Read — อ่านแถวเดิมซ้ำแล้วได้ค่าต่างกัน
- Phantom Read — query เงื่อนไขเดิมซ้ำแล้วได้ชุดแถวต่างกัน
- Lost Update — update หนึ่งเขียนทับผลของอีก transaction
- Write Skew — แต่ละ transaction ดูถูกต้อง แต่ผลรวมผิด business rule

Atomic update มักปลอดภัยกว่าการอ่านมาคำนวณใน Go:

```sql
UPDATE accounts
SET balance = balance + $1
WHERE id = $2
RETURNING *;
```

### Transactions in Go

`execTx` รวม boilerplate ของ transaction ไว้จุดเดียว:

```go
func (store *SQLStore) execTx(
    ctx context.Context,
    fn func(*Queries) error,
) error {
    tx, err := store.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }

    q := store.Queries.WithTx(tx)
    if err := fn(q); err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("tx err: %w, rollback err: %v", err, rbErr)
        }
        return err
    }

    return tx.Commit()
}
```

ทุก query ใน callback ต้องเรียกผ่าน `q` เพราะ `q` ผูกกับ transaction:

```go
store.execTx(ctx, func(q *Queries) error {
    _, err := q.CreateTransfer(ctx, arg)
    return err
})
```

ไม่ควรเรียกผ่าน `store` ภายใน callback เพราะอาจออกนอก transaction:

```go
store.CreateTransfer(ctx, arg) // ไม่แนะนำใน callback
```

Query เดียวไม่จำเป็นต้องเปิด explicit transaction เสมอ PostgreSQL ทำให้ statement เดียวเป็น atomic อยู่แล้ว ใช้ transaction เมื่อหลาย statements ต้อง commit หรือ rollback พร้อมกัน

### Deadlock

Deadlock เกิดเมื่อ transactions รอ lock กันเป็นวงจร:

```text
Tx A ถือ Account 1 และรอ Account 2
Tx B ถือ Account 2 และรอ Account 1
```

PostgreSQL จะยกเลิก transaction หนึ่งตัวและคืน SQLSTATE `40P01`

แนวทางลด deadlock:

1. ล็อกหลาย rows ตามลำดับเดียวกัน เช่น ID น้อยก่อน
2. ใช้ atomic update เมื่อทำได้
3. ทำ transaction ให้สั้น
4. ไม่เรียก external service ขณะถือ lock
5. ใช้ lock ที่เหมาะสม เช่น `FOR NO KEY UPDATE` เมื่อไม่ได้แก้ key
6. Retry ทั้ง transaction โดยจำกัดจำนวนครั้งและตรวจ Context

`40P01` คือ deadlock ส่วน `40001` คือ serialization failure ทั้งสองกรณีอาจ retry ทั้ง transaction ได้

## 4. sqlc

sqlc อ่าน schema และ SQL queries แล้วสร้าง Go code แบบ type-safe

เราเขียน SQL:

```sql
-- name: GetAccount :one
SELECT * FROM accounts
WHERE id = $1;
```

sqlc สร้าง method:

```go
func (q *Queries) GetAccount(
    ctx context.Context,
    id int64,
) (Account, error)
```

ข้อดี:

- ลด boilerplate เช่น `QueryRow`, `Scan` และ params structs
- ตรวจ column และ type ตั้งแต่ generate
- ยังเขียนและปรับ SQL ได้โดยตรง
- ได้ models และ methods ที่ type-safe

รัน:

```bash
sqlc generate
# หรือ
make sqlc
```

ไฟล์ที่มีข้อความนี้ห้ามแก้เอง:

```go
// Code generated by sqlc. DO NOT EDIT.
```

### Current `sqlc.yaml` Configuration

```yaml
emit_json_tags: true
emit_prepared_queries: false
emit_interface: true
emit_exact_table_names: false
emit_empty_slices: true
```

- `emit_json_tags` เพิ่ม JSON tags ให้ models
- `emit_prepared_queries` ไม่สร้าง explicit prepared statements
- `emit_interface` สร้าง `Querier` interface
- `emit_exact_table_names` ให้ sqlc singularize ชื่อ model
- `emit_empty_slices` ให้ query แบบ `:many` คืน slice ว่างแทน `nil`

`sqlc generate` ไม่ได้สร้าง database, รัน migration หรือเปิด connection สิ่งเหล่านี้เป็นหน้าที่ของ application และ migration tool

## 5. Store, Interfaces and Dependency Injection

### Queries vs Store

`Queries` ทำงานระดับ SQL query เดียว ส่วน `Store` เพิ่ม business transaction เช่น `TransferTx`

```text
Queries → GetAccount, CreateEntry, AddAccountBalance
Store   → Queries ทั้งหมด + TransferTx
```

โค้ดปัจจุบัน:

```go
type Store interface {
    Querier
    TransferTx(ctx context.Context, arg TransferTxParams) (TransferResult, error)
}

type SQLStore struct {
    *Queries
    db *sql.DB
}
```

- `Store` คือ contract
- `SQLStore` คือ implementation ที่ใช้ PostgreSQL
- Embedded `*Queries` ทำให้ `SQLStore` มี methods จาก `Querier`
- `db *sql.DB` ใช้เปิด transaction
- `execTx` เป็น private helper จึงไม่ต้องอยู่ใน interface

Constructor สร้าง `SQLStore` แล้วคืนในรูป `Store`:

```go
func NewStore(db *sql.DB) Store {
    return &SQLStore{
        db:      db,
        Queries: New(db),
    }
}
```

Interface กำหนด methods ไม่ได้กำหนด fields ดังนั้น `SQLStore` มี `*sql.DB` ได้แม้ `Store` ไม่มี field นี้

### Implicit Interface Implementation

Go ไม่ต้องเขียนคำว่า `implements` ถ้า type มี methods ครบก็ implement interface อัตโนมัติ

Compile-time assertion:

```go
var _ Querier = (*Queries)(nil)
```

แปลว่าให้ compiler ตรวจว่า `*Queries` implement `Querier` ครบ โดยไม่สร้าง object จริง

#### Interface เก็บ Concrete Type

```go
maker, err := NewPasetoMaker(key)
```

แม้ตัวแปร `maker` มี static type เป็น `Maker` แต่ค่าจริงภายในมี dynamic type เป็น `*PasetoMaker` เพราะ constructor คืน `&PasetoMaker{...}` เมื่อเรียก `maker.VerifyToken()` Go จึงเลือก method ของ `PasetoMaker` ให้อัตโนมัติ

```text
static type  = Maker
dynamic type = *PasetoMaker
```

ปกติไม่ต้องตรวจ concrete type เพราะจุดประสงค์ของ interface คือให้ JWTMaker และ PasetoMaker ใช้งานผ่าน methods ชุดเดียวกัน

### Dependency Injection

`Server` รับ Store จากภายนอก:

```go
type Server struct {
    store  db.Store
    router *gin.Engine
}

func NewServer(store db.Store) *Server {
    return &Server{store: store}
}
```

Production ส่ง `SQLStore`:

```go
store := db.NewStore(conn)
server := api.NewServer(store)
```

Unit test ส่ง `MockStore`:

```go
store := mockdb.NewMockStore(ctrl)
server := api.NewServer(store)
```

ไม่ใช้ `*db.Store` เพราะ `Store` เป็น interface ที่เก็บ pointer implementation เช่น `*SQLStore` อยู่ภายในแล้ว Pointer to interface เพิ่มอีกชั้นโดยไม่จำเป็น

## 6. Common Go Basics

### Boilerplate

Boilerplate คือโค้ดที่จำเป็นแต่มีรูปแบบซ้ำ เช่นเปิด query, scan fields และส่ง error เครื่องมืออย่าง sqlc และ mockgen ช่วยสร้างโค้ดส่วนนี้

### `int` vs `int64`

- `int` เหมาะกับ index, `len()` และ loop
- `int64` มีขนาด 64 บิตแน่นอน เหมาะกับ ID, balance และ PostgreSQL `BIGINT`
- Go ไม่แปลง type ให้อัตโนมัติ

```go
var count int = 10
var balance int64 = int64(count)
```

### `fmt` and Format Verbs

`fmt` ใช้แสดงผล สร้างข้อความ และสร้าง error เช่น `Println`, `Printf`, `Sprintf` และ `Errorf`

| รูปแบบ | ใช้กับ |
| --- | --- |
| `%s` | string |
| `%q` | string พร้อม quote |
| `%d` | integer ฐานสิบ |
| `%f` | float เช่น `%.2f` แสดง 2 ตำแหน่ง |
| `%t` | boolean |
| `%v` / `%+v` | ค่าทั่วไป / struct พร้อมชื่อ field |
| `%T` | type ของค่า |
| `%p` | address ของ pointer |
| `%%` | เครื่องหมาย `%` |
| `%w` | wrap error ใช้กับ `fmt.Errorf` |

```go
return fmt.Errorf("failed to hash password: %w", err)
```

`%w` เพิ่มคำอธิบายและยังเก็บ error เดิมไว้ให้ตรวจด้วย `errors.Is` หรือ `errors.As` ได้

#### `errors.New()` กับ `fmt.Errorf()`

- `errors.New("...")` ใช้สร้าง error ที่เป็นข้อความคงที่ เช่น `ErrInvalidToken`
- `fmt.Errorf("... %d", value)` ใช้เมื่อจำเป็นต้องแทรกค่าลงในข้อความ
- `fmt.Errorf("...: %w", err)` ใช้เพิ่มคำอธิบายและห่อ error เดิม โดยยังตรวจด้วย `errors.Is` ได้

```go
var ErrInvalidToken = errors.New("token is invalid")

return fmt.Errorf("secret key must have at least %d characters", minSize)
return fmt.Errorf("create token: %w", err)
```

จำง่าย ๆ: ข้อความคงที่ใช้ `errors.New`; มีตัวแปรหรือห่อ error เดิมใช้ `fmt.Errorf`

### Pointer

Pointer เก็บที่อยู่ของค่าอื่น:

```go
x := 10
p := &x
*p = 20 // x กลายเป็น 20
```

- `&x` ขอที่อยู่ของ `x`
- `*int` คือ type pointer ไปยัง `int`
- `*p` เข้าถึงค่าที่ pointer ชี้อยู่
- Dereference `nil` จะ panic

Constructor คืน pointer เมื่อ object ควรใช้งานเป็น instance เดิม:

```go
func NewQueries(db DBTX) *Queries
```

ดังนั้น:

```go
store := NewStore(db) // Go อนุมาน return type ให้
```

### Method Receiver

```go
func (q *Queries) GetAccount(...) (...)
```

- `q` คือ receiver variable
- `*Queries` คือ type เจ้าของ method
- Pointer receiver ช่วยทำงานกับ object เดิมและไม่ copy struct

#### Value Receiver กับ Pointer Receiver

Value receiver ใช้ type ปกติ เช่น `(a Account)` โดย method จะได้รับสำเนาของ struct การแก้ field ภายใน method จึงไม่กระทบค่าต้นฉบับ:

```go
func (a Account) Rename(name string) {
    a.Owner = name // แก้เฉพาะสำเนา
}
```

Pointer receiver ใช้ pointer type เช่น `(a *Account)` ทำให้ method เข้าถึงและแก้ struct ตัวเดิมได้:

```go
func (a *Account) Rename(name string) {
    a.Owner = name // แก้ Account ตัวจริง
}
```

สรุปสั้น ๆ:

- `(a Account)` คือ value receiver เหมาะเมื่อไม่ต้องแก้ค่าต้นฉบับ
- `(a *Account)` คือ pointer receiver เหมาะเมื่อต้องแก้ค่าต้นฉบับ หรือไม่ต้องการ copy struct
- Go ช่วยหา address ให้อัตโนมัติ จึงเรียก `account.Rename("Tom")` ได้ แม้ method จะใช้ pointer receiver

### Embedded Field

```go
type SQLStore struct {
    *Queries
}
```

ทำให้เรียก promoted methods ได้:

```go
store.GetAccount(ctx, id)
```

เทียบเท่ากับ:

```go
store.Queries.GetAccount(ctx, id)
```

### Goroutines and Channels

Goroutine ใช้รันงานพร้อมกัน:

```go
go doWork()
```

Channel ใช้ส่งข้อมูลหรือประสานงาน:

```go
results := make(chan Result)
results <- result
result := <-results
```

Concurrent tests ช่วยเปิดเผย race condition และ deadlock ที่ sequential tests อาจไม่พบ

### `defer`

`defer` ทำงานก่อน function ที่ประกาศมันจะจบ ไม่ใช่ตอนจบโปรเจกต์

```go
func readFile() error {
    file, err := os.Open("data.txt")
    if err != nil {
        return err
    }
    defer file.Close()

    return nil
}
```

หลาย `defer` ทำงานย้อนลำดับแบบ LIFO และ `os.Exit()`/`log.Fatal()` จะไม่รัน `defer`

## 7. Context

`context.Context` ใช้ส่ง cancellation, timeout, deadline และ request metadata ผ่าน function หลายชั้น

```text
HTTP Request → Handler → Service → Store → Database
```

รับ Context เป็น parameter ตัวแรกและส่งตัวเดิมต่อ:

```go
func GetAccount(ctx context.Context, id int64) (Account, error) {
    return store.GetAccount(ctx, id)
}
```

### Creating Contexts

```go
context.Background()               // จุดเริ่มต้น
context.WithCancel(parent)         // ยกเลิกเองได้
context.WithTimeout(parent, d)     // หมดอายุหลังระยะเวลา
context.WithDeadline(parent, time) // หมดอายุตามเวลาที่กำหนด
```

เมื่อสร้าง cancellable context ต้องเรียก `cancel()`:

```go
ctx, cancel := context.WithTimeout(parent, 2*time.Second)
defer cancel()
```

### Understanding `ctx.Done()`

ให้นึกว่า `ctx.Done()` คือกริ่งเลิกงาน:

```text
ยังไม่ cancel → กริ่งยังไม่ดัง → ผู้รอยังรอ
cancel()      → Done channel ปิด → ผู้รอรู้ว่าควรหยุด
```

```go
func waitForResult(ctx context.Context, resultCh <-chan string) (string, error) {
    select {
    case <-ctx.Done():
        return "", ctx.Err()
    case result := <-resultCh:
        return result, nil
    }
}
```

การปิด channel แจ้งผู้รอหลาย goroutines พร้อมกันได้ แต่ไม่ได้บังคับฆ่า goroutine งานต้องตรวจ `Done()` หรือเรียก API ที่รองรับ Context

### `ctx.JSON()` in Gin

`*gin.Context` เป็นตัวกลางของ HTTP request รอบนั้น เมื่อเรียก:

```go
ctx.JSON(http.StatusBadRequest, errorResponse(err))
return
```

- `ctx.JSON(...)` ใช้ ResponseWriter ภายใน Context เขียน JSON และ status กลับไปยัง client
- Response ไม่ได้ถูกเก็บแล้วส่งกลับไปพร้อม Context
- `ctx.JSON(...)` ไม่หยุด function จึงต้องใช้ `return` เมื่อไม่ต้องการให้ handler ทำงานต่อ
- ผู้รับ Response คือ client ที่ส่ง request เช่น Browser, Frontend หรือ Postman

กฎสำคัญ:

- ใช้ Context จาก request เช่น `r.Context()`
- ใช้ `Background()` ใน `main`, setup หรือ test
- อย่าส่ง `nil`
- โดยทั่วไปไม่เก็บ Context ใน struct
- ใช้ `WithValue` เฉพาะ metadata เช่น request ID ไม่ใช้แทน business parameters

## 8. Testing

### `t *testing.T`

`t` คือตัวแทนของ Test ที่กำลังรัน โดย Go testing framework เป็นผู้สร้างและส่งเข้ามา:

```go
func createRandomUser(t *testing.T) User {
    require.NoError(t, err)
    return user
}
```

ส่ง `t` เข้า helper เพื่อให้ `require`, `t.Error()` หรือ `t.Fatal()` รายงานและหยุด Test ที่เรียกอยู่ได้

```text
*testing.T     → ควบคุมและรายงานผล Test
context.Context → cancellation, timeout และ deadline ของงาน
```

ดังนั้น `t *testing.T` ไม่ใช่ Context

### `TestMain`

ใช้ setup และ cleanup resource ของ tests ทั้ง package:

```go
func TestMain(m *testing.M) {
    // เปิด database
    code := m.Run()
    // ปิด database
    os.Exit(code)
}
```

`m.Run()` รัน tests และคืน exit code:

```text
0        → ผ่าน
ไม่ใช่ 0 → ไม่ผ่าน
```

ต้อง cleanup ก่อน `os.Exit()` เพราะ `os.Exit()` ไม่รัน `defer`

### Unit Tests vs Integration Tests

| แบบ | ใช้อะไร | ตรวจอะไร |
| --- | --- | --- |
| Unit Test | Mock/Fake | validation, business logic, HTTP response |
| Integration Test | PostgreSQL จริง | SQL, schema, scan และ transaction |

ระบบควรมีทั้งสองแบบเพราะตรวจคนละส่วน

### Table-driven Tests

รวมหลายสถานการณ์ในตารางเดียว:

```go
testCases := []struct {
    name          string
    buildStubs    func(*mockdb.MockStore)
    checkResponse func(*httptest.ResponseRecorder)
}{
    // OK, NotFound, InternalError, InvalidID
}
```

รันเป็น subtests:

```go
for _, tc := range testCases {
    t.Run(tc.name, func(t *testing.T) {
        // setup, execute, verify
    })
}
```

## 9. GoMock and HTTP Handler Testing

คำสั่งนี้อ่าน `Store` interface จาก `db/sqlc` แล้วสร้าง `MockStore` สำหรับ Unit Test โดยไม่ต้องต่อ Database จริง:

```bash
mockgen \
  -package mockdb \
  -destination db/mock/store.go \
  github.com/MumAroi/go-simplebank/db/sqlc \
  Store
```

- `-package` กำหนดชื่อ package ในไฟล์ที่สร้าง
- `-destination` กำหนดไฟล์ปลายทาง
- import path ชี้ไปยัง package ต้นฉบับ
- `Store` คือ interface ที่ต้องการ mock รวมถึง methods จาก `Querier` ที่ฝังอยู่

ไฟล์ generated ห้ามแก้เอง หลัง interface เปลี่ยนให้รัน `make mock` ใหม่

### Defining Mock Behavior

```go
store.EXPECT().
    GetAccount(gomock.Any(), gomock.Eq(account.ID)).
    Times(1).
    Return(account, nil)
```

ความหมาย:

- คาดว่า `GetAccount` ถูกเรียกหนึ่งครั้ง
- Context เป็นค่าใดก็ได้
- ID ต้องตรง
- คืน `account, nil`

Error cases ใช้ input เดิมแต่กำหนด return ต่างกันเพื่อทดสอบ handler ทีละเงื่อนไข:

```text
sql.ErrNoRows  → 404 Not Found
sql.ErrConnDone → 500 Internal Server Error
```

### `httptest`

```go
recorder := httptest.NewRecorder()
request := httptest.NewRequest(http.MethodGet, "/accounts/1", nil)

server.router.ServeHTTP(recorder, request)
```

- `ResponseRecorder` เก็บ status, headers และ body
- Request และ handler ทำงานใน memory ไม่ต้องเปิด network port
- Router และ handler เป็นของจริง แต่ Store เป็น mock

ตรวจ response:

```go
require.Equal(t, http.StatusOK, recorder.Code)

data, err := io.ReadAll(recorder.Body)
require.NoError(t, err)

var got db.Account
require.NoError(t, json.Unmarshal(data, &got))
require.Equal(t, account, got)
```

`io.ReadAll` ใช้แทน `ioutil.ReadAll` ที่ deprecated แล้ว

## 10. Coverage

```bash
go test -v -cover ./...
```

- `PASS` หมายถึง assertions ที่รันไม่ล้มเหลว
- Coverage คือเปอร์เซ็นต์ statements ที่ tests วิ่งผ่าน
- Tests ผ่านหมดไม่ได้แปลว่า coverage ต้องเป็น 100%
- Coverage สูงไม่ได้รับประกันว่า test ตรวจ behavior ได้ดี

สร้างรายงาน:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go tool cover -html=coverage.out
```

`./...` คือ package ปัจจุบันและ packages ใต้ directory ทั้งหมด เป็น pattern ของ Go tools ไม่ใช่ wildcard ทั่วไปของ shell

## 11. GitHub Actions and CI

Workflow ปัจจุบันทำงานเมื่อ push หรือเปิด Pull Request เข้า `main`:

```text
Checkout code
    ↓
Setup Go
    ↓
เริ่ม PostgreSQL service
    ↓
ติดตั้ง migrate
    ↓
รัน migrations
    ↓
รัน tests
```

PostgreSQL service เป็น database ชั่วคราวและถูกลบเมื่อ job จบ

ข้อควรรู้:

- `services.postgres.env` ถูกใช้ตอนสร้าง container ก่อน `steps`
- จึงอ่าน `.env` ที่สร้างใน step ภายหลังโดยตรงไม่ได้
- ค่า CI ทั่วไปใช้ workflow `env` หรือ GitHub Variables
- Password จริงใช้ GitHub Secrets
- การ copy `.env.example` เป็น `.env` ไม่ได้ export environment variables อัตโนมัติ

## 12. Common Commands

```bash
# Database
make createdb
make dropdb
make migrateup
make migratedown

# Generate code
make sqlc
make mock

# Tests
make test
go test ./db/sqlc
go test ./api
go test ./... -cover

# Application
make server
```

ติดตั้ง tools:

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
go install go.uber.org/mock/mockgen@v0.6.0
go mod tidy
```

## Key Takeaways

```text
Migration ดูแล schema
sqlc สร้าง type-safe query code
Store เป็น contract ระหว่าง API กับ database
SQLStore ใช้ PostgreSQL จริง
execTx รวมหลาย queries เป็น transaction
Context ควบคุมอายุของ request
MockStore ใช้ใน unit test
PostgreSQL จริงใช้ใน integration test
CI รัน migrations และ tests อัตโนมัติ
```

แก่นของการออกแบบคือให้แต่ละชั้นรับผิดชอบหน้าที่ของตัวเอง และเชื่อมกันผ่าน interface ที่ทดสอบหรือเปลี่ยน implementation ได้ง่าย

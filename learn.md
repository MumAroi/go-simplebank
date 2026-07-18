# Database Migration

## การสร้างไฟล์ Migration

ใช้คำสั่งต่อไปนี้เพื่อสร้างไฟล์ migration สำหรับโครงสร้างฐานข้อมูลเริ่มต้น:

```bash
migrate create -ext sql -dir db/migration -seq init_schema
```

ความหมายของแต่ละส่วน:

- `migrate create` สั่ง CLI ของ `golang-migrate` ให้สร้าง migration ใหม่
- `-ext sql` กำหนดนามสกุลของไฟล์เป็น `.sql`
- `-dir db/migration` กำหนดให้สร้างไฟล์ในโฟลเดอร์ `db/migration`
- `-seq` ใช้เลขลำดับเป็นเวอร์ชัน เช่น `000001`, `000002`
- `init_schema` คือชื่อ migration ซึ่งสื่อว่าใช้สร้างโครงสร้างฐานข้อมูลเริ่มต้น

เมื่อรันคำสั่ง จะได้ไฟล์สองไฟล์:

```text
db/migration/
├── 000001_init_schema.up.sql
└── 000001_init_schema.down.sql
```

- `000001_init_schema.up.sql` ใช้สำหรับนำการเปลี่ยนแปลงไปใช้ เช่น `CREATE TABLE`
- `000001_init_schema.down.sql` ใช้สำหรับย้อนกลับการเปลี่ยนแปลง เช่น `DROP TABLE`

ตัวอย่างเนื้อหาไฟล์ `up.sql`:

```sql
CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    owner VARCHAR NOT NULL,
    balance BIGINT NOT NULL
);
```

ตัวอย่างเนื้อหาไฟล์ `down.sql`:

```sql
DROP TABLE accounts;
```

สรุป: คำสั่งนี้สร้าง migration ชื่อ `init_schema` พร้อมไฟล์สำหรับดำเนินการเปลี่ยนแปลง (`up`) และย้อนกลับการเปลี่ยนแปลง (`down`)

## การตั้งค่า sqlc

ไฟล์ `sqlc.yaml` ใช้บอก `sqlc` ว่าจะอ่าน schema และ SQL query จากที่ใด รวมถึงกำหนดรูปแบบของ Go code ที่ต้องการสร้าง:

```yaml
version: "2"

sql:
  - name: "db"
    engine: "postgresql"
    schema: "./db/migration"
    queries: "./db/query"
    gen:
      go:
        package: "db"
        out: "./db/sqlc"
        sql_package: "database/sql"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: false
        emit_exact_table_names: false
```

ความหมายของค่าหลัก:

- `engine` กำหนดชนิดฐานข้อมูล ในที่นี้คือ PostgreSQL
- `schema` ระบุตำแหน่ง schema หรือโฟลเดอร์ migration โดย `sqlc` จะอ่านไฟล์ migration ตามลำดับชื่อ
- `queries` ระบุโฟลเดอร์ที่เก็บ SQL query สำหรับสร้าง Go methods
- `package` กำหนดชื่อ package ของ Go code ที่สร้างขึ้น
- `out` กำหนดโฟลเดอร์ปลายทางของ Go code
- `sql_package` กำหนด database package ที่ใช้ ในที่นี้คือ `database/sql`

### ตัวเลือกการสร้าง Go code

#### `emit_json_tags: true`

เพิ่ม JSON tags ให้ field ของ struct เพื่อให้ใช้งานกับ JSON API ได้สะดวก:

```go
type Account struct {
    ID      int64 `json:"id"`
    Balance int64 `json:"balance"`
}
```

#### `emit_prepared_queries: false`

ไม่สร้าง prepared statements ไว้ล่วงหน้า แต่ให้ query เรียกฐานข้อมูลโดยตรงตามปกติ หากตั้งเป็น `true` จะมีขั้นตอนเตรียมและเก็บ statements สำหรับนำกลับมาใช้ซ้ำ

#### `emit_interface: false`

ไม่สร้าง interface ที่รวม query methods หากตั้งเป็น `true` จะสร้าง interface เช่น `Querier` ซึ่งช่วยในการทำ mock, dependency injection และ unit test

```go
type Querier interface {
    GetAccount(ctx context.Context, id int64) (Account, error)
    ListAccounts(ctx context.Context) ([]Account, error)
}
```

#### `emit_exact_table_names: false`

อนุญาตให้ `sqlc` เปลี่ยนชื่อตารางแบบพหูพจน์เป็นชื่อ model แบบเอกพจน์ เช่น ตาราง `accounts` จะสร้าง struct ชื่อ `Account` หากตั้งเป็น `true` ชื่อ struct จะอิงชื่อตารางตรง ๆ เป็น `Accounts`

หลังจากสร้างไฟล์ query ใน `db/query` แล้ว ให้สร้าง Go code ด้วยคำสั่ง:

```bash
sqlc generate
```

## หลังจากรัน `sqlc generate` เกิดอะไรขึ้น

`sqlc generate` จะอ่าน schema และ SQL query แล้วสร้าง Go code สำหรับติดต่อฐานข้อมูลโดยอัตโนมัติ:

```text
Migration ──> วิเคราะห์ Schema ──┐
                                 ├──> สร้าง Go code
SQL Query ──> วิเคราะห์ Query ───┘
```

### 1. อ่าน Database Schema

`sqlc` อ่าน migration ใน `db/migration` เพื่อทำความเข้าใจตาราง คอลัมน์ ชนิดข้อมูล ค่า nullable, Primary Key และ Foreign Key จากนั้นจึงแปลงชนิดข้อมูล PostgreSQL เป็น Go เช่น:

| PostgreSQL | Go |
| --- | --- |
| `BIGINT` / `BIGSERIAL` | `int64` |
| `VARCHAR` / `TEXT` | `string` |
| `BOOLEAN` | `bool` |
| `TIMESTAMPTZ` | `time.Time` |
| nullable `TEXT` | `sql.NullString` |

### 2. อ่าน SQL Query

ตัวอย่างไฟล์ `db/query/account.sql`:

```sql
-- name: GetAccount :one
SELECT * FROM accounts
WHERE id = $1 LIMIT 1;
```

- `GetAccount` คือชื่อ Go method ที่จะถูกสร้าง
- `:one` หมายถึงคืนผลลัพธ์หนึ่งแถว
- `$1` คือ parameter ตัวแรก โดย `sqlc` จะอนุมานชนิดจาก schema

รูปแบบผลลัพธ์ที่ใช้บ่อย ได้แก่ `:one` สำหรับหนึ่งแถว, `:many` สำหรับหลายแถว, `:exec` สำหรับคำสั่งที่ไม่คืน record และ `:execrows` สำหรับคืนจำนวนแถวที่ได้รับผลกระทบ

### 3. ตรวจสอบ Query

`sqlc` ตรวจสอบว่าตารางและคอลัมน์มีอยู่จริง พร้อมวิเคราะห์ชนิดของ parameter และผลลัพธ์ หาก query ไม่สอดคล้องกับ schema คำสั่งจะหยุดและแจ้ง error ก่อนรัน application

### 4. สร้าง Models และ Query Methods

`db/sqlc/models.go` จะมี struct ที่แทนข้อมูลในตาราง:

```go
type Account struct {
    ID      int64  `json:"id"`
    Owner   string `json:"owner"`
    Balance int64  `json:"balance"`
}
```

ส่วน query จะถูกสร้างเป็น Go method ที่มีชนิดข้อมูลชัดเจน เช่น:

```go
func (q *Queries) GetAccount(ctx context.Context, id int64) (Account, error)
```

จึงไม่ต้องเขียน `QueryRow`, `Scan` และการแปลงชนิดข้อมูลซ้ำด้วยตนเอง

### 5. ไฟล์ที่ถูกสร้าง

```text
db/sqlc/
├── db.go
├── models.go
└── account.sql.go
```

- `db.go` เก็บ `Queries`, `DBTX` และ constructor `New`
- `models.go` เก็บ struct ที่สร้างจากตาราง
- `account.sql.go` เก็บ SQL constants และ methods จาก `account.sql`

ตัวอย่างการใช้งาน generated code:

```go
database, err := sql.Open("postgres", databaseURL)
if err != nil {
    log.Fatal(err)
}

queries := db.New(database)
account, err := queries.GetAccount(context.Background(), 1)
```

### สิ่งที่ `sqlc generate` ไม่ได้ทำ

คำสั่งนี้ไม่ได้สร้างฐานข้อมูลหรือตารางจริง ไม่ได้รัน migration ไม่ได้แก้ไขข้อมูล และไม่ได้เปิด database connection การนำ schema ไปใช้กับ PostgreSQL จริงยังต้องใช้ migration tool เช่น `golang-migrate`

สรุป: `sqlc generate` แปลง schema และ SQL query ให้เป็น Go code แบบ type-safe ช่วยลด boilerplate และตรวจพบ query ที่ไม่สอดคล้องกับ schema ตั้งแต่ขั้นตอน generate

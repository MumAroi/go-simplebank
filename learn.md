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

## Method Receiver ของ Go เทียบกับ TypeScript

โค้ดที่ `sqlc` สร้างมีลักษณะดังนี้:

```go
func (q *Queries) CreateAccount(
    ctx context.Context,
    arg CreateAccountParams,
) (Account, error) {
    // ใช้งาน q.db เพื่อ query ฐานข้อมูล
}
```

ส่วน `(q *Queries)` เรียกว่า **method receiver** หมายความว่า `CreateAccount` เป็น method ของชนิด `*Queries`

- `q` คือชื่อตัวแปรที่ใช้อ้างถึง object ภายใน method
- `*Queries` คือ pointer ไปยังชนิด `Queries` ที่เป็นเจ้าของ method
- `CreateAccount` คือชื่อ method

เมื่อเรียก:

```go
testQueries.CreateAccount(ctx, arg)
```

Go จะนำ `testQueries` มาเป็นค่า `q` ให้อัตโนมัติ ภายใน method จึงเข้าถึง database connection ได้ด้วย `q.db`

### เปรียบเทียบกับ TypeScript

Go:

```go
type Queries struct {
    db DBTX
}

func New(db DBTX) *Queries {
    return &Queries{db: db}
}

func (q *Queries) CreateAccount(
    ctx context.Context,
    arg CreateAccountParams,
) (Account, error) {
    // q.db ทำหน้าที่ query ฐานข้อมูล
}
```

TypeScript ที่มีแนวคิดใกล้เคียงกัน:

```ts
class Queries {
  constructor(private db: DBTX) {}

  async createAccount(
    arg: CreateAccountParams
  ): Promise<Account> {
    // this.db ทำหน้าที่ query ฐานข้อมูล
  }
}
```

การใช้งาน:

```go
// Go
testQueries := New(conn)
account, err := testQueries.CreateAccount(ctx, arg)
```

```ts
// TypeScript
const testQueries = new Queries(conn);
const account = await testQueries.createAccount(arg);
```

สิ่งที่เทียบกันได้:

| Go | TypeScript |
| --- | --- |
| `type Queries struct` | `class Queries` |
| `db DBTX` | `private db: DBTX` |
| `New(conn)` | `new Queries(conn)` |
| `(q *Queries)` | `this` ภายใน class |
| `q.db` | `this.db` |
| `CreateAccount(...)` | `createAccount(...)` |
| `(Account, error)` | `Promise<Account>` และ error/exception |

`New` ใน Go เป็น factory function ตามธรรมเนียม ไม่ใช่ constructor พิเศษของภาษาเหมือน `constructor` ใน TypeScript

สรุป: `q` ใน method receiver ทำหน้าที่ใกล้เคียงกับ `this` ใน TypeScript และทำให้ method สามารถเข้าถึงข้อมูลภายใน `Queries` เช่น database connection ได้

## การรัน Go Tests

รัน test ทั้งหมดใน package `db/sqlc` ด้วยคำสั่ง:

```bash
go test -timeout 30s ./db/sqlc
```

- `go test` compile และรัน test
- `-timeout 30s` หยุดการทดสอบหากใช้เวลารวมเกิน 30 วินาที
- `./db/sqlc` คือ package ที่ต้องการทดสอบ

เพิ่ม `-v` เพื่อแสดงชื่อและผลของแต่ละ test อย่างละเอียด:

```bash
go test -v -timeout 30s ./db/sqlc
```

### การเลือก Test ด้วย `-run`

ตัวเลือก `-run` รับ Regular Expression สำหรับกรองชื่อ test ที่ต้องการรัน

รันเฉพาะ `TestCreateAccount`:

```bash
go test -timeout 30s ./db/sqlc -run '^TestCreateAccount$'
```

รันทุก test ที่ชื่อขึ้นต้นด้วย `TestAccount`:

```bash
go test -timeout 30s ./db/sqlc -run '^TestAccount'
```

ใน Regular Expression เครื่องหมาย `^` หมายถึงจุดเริ่มต้น และ `$` หมายถึงจุดสิ้นสุด ดังนั้น `^TestCreateAccount$` จะตรงกับชื่อนี้แบบพอดีเท่านั้น

### หน้าที่ของ `TestMain`

`TestMain` เป็น lifecycle hook ของ test ทั้ง package เหมาะสำหรับ setup และ cleanup ทรัพยากรที่ tests ใช้ร่วมกัน เช่น database connection:

```go
var testQueries *Queries

func TestMain(m *testing.M) {
    conn, err := sql.Open(dbDriver, dbSource)
    if err != nil {
        log.Fatal("cannot connect to db:", err)
    }

    testQueries = New(conn)
    os.Exit(m.Run())
}
```

Go จะเรียก `TestMain` อัตโนมัติเมื่อทดสอบ package นี้ จึงไม่ต้องรันด้วย:

```bash
go test ./db/sqlc -run '^TestMain$'
```

`-run` ใช้กรอง test functions ปกติ แต่ `TestMain` เป็น lifecycle hook ไม่ใช่ test case แม้ `TestMain` จะถูกเรียก แต่ `m.Run()` อาจไม่พบ test case ที่ตรงกับ filter และแสดง `no tests to run`

### `m.Run()` และ `os.Exit()`

```go
os.Exit(m.Run())
```

ทำงานเป็นลำดับดังนี้:

1. `m.Run()` รัน test cases ทั้งหมดที่ผ่านตัวกรอง และคืน exit code
2. `os.Exit(...)` จบ process พร้อมส่ง exit code กลับไปยัง command line หรือ CI

- exit code `0` หมายถึง test ผ่าน
- exit codeที่ไม่ใช่ `0` หมายถึง test ล้มเหลว

ข้อควรระวังคือ `os.Exit` จบ process ทันที ทำให้ `defer` ไม่ถูกเรียก หากต้อง cleanup ให้ทำก่อน `os.Exit`:

```go
func TestMain(m *testing.M) {
    // setup
    code := m.Run()
    conn.Close()
    os.Exit(code)
}
```

### ทำไม `testQueries` เรียก `CreateAccount` ได้

`New(conn)` คืนค่า `*Queries` และเก็บ `conn` ไว้ใน field `db`:

```go
func New(db DBTX) *Queries {
    return &Queries{db: db}
}
```

เมื่อกำหนด:

```go
testQueries = New(conn)
```

ตัวแปร `testQueries` จึงมีชนิด `*Queries` และสามารถเรียกทุก method ที่มี receiver เป็น `*Queries` ได้ เช่น:

```go
func (q *Queries) CreateAccount(...) (...)
```

```go
account, err := testQueries.CreateAccount(ctx, arg)
```

ระหว่างทำงาน `q` คือ `testQueries` และ `q.db` คือ database connection ที่ส่งเข้า `New(conn)`

### ทำไม `New` รับ `*sql.DB` เป็น `DBTX` ได้

`DBTX` เป็น interface ที่ประกาศ database methods ที่ generated queries ต้องใช้ เช่น `ExecContext`, `QueryContext` และ `QueryRowContext` ส่วน `*sql.DB` มี methods เหล่านี้ครบ จึง implement `DBTX` โดยอัตโนมัติตามระบบ implicit interface ของ Go:

```go
conn, err := sql.Open(dbDriver, dbSource) // conn มีชนิด *sql.DB
testQueries = New(conn)                  // ส่งเป็น DBTX ได้
```

ข้อดีคือ `Queries` ไม่ผูกติดกับ `*sql.DB` เพียงชนิดเดียว และสามารถใช้ implementation อื่นที่มี methods ครบ เช่น `*sql.Tx` สำหรับ transaction ได้

### Error: `expected 'package', found 'EOF'`

หากพบ error ลักษณะนี้:

```text
account_test.go:1:1: expected 'package', found 'EOF'
```

แปลว่าไฟล์ Go ว่างหรือไม่มีคำประกาศ `package` เพราะ Go ทุกไฟล์ต้องระบุ package อย่างน้อย:

```go
package db
```

หากยังไม่ได้ใช้งานไฟล์ test นั้น ให้เพิ่ม package declaration หรือเอาไฟล์ว่างออกก่อนรัน tests

## การใช้ Testify ใน Go Tests

`testify` เป็นไลบรารีเสริมสำหรับเขียน assertions ให้สั้นและอ่านง่ายขึ้น แต่ไม่จำเป็นต้องใช้ เพราะ standard library `testing` ของ Go สามารถใช้เขียน test ได้ครบถ้วนอยู่แล้ว

ติดตั้งด้วยคำสั่ง:

```bash
go get github.com/stretchr/testify
```

คำสั่งนี้จะเพิ่ม dependency ลงใน `go.mod` และข้อมูล checksum ลงใน `go.sum` หากใช้เพียง package `require` สามารถระบุ package โดยตรงได้:

```bash
go get github.com/stretchr/testify/require
```

Go ไม่มี `devDependencies` แยกเหมือน Node.js ดังนั้น dependency ที่ใช้เฉพาะ test ก็จะอยู่ใน `go.mod` ตามปกติ

### ใช้ `testing` อย่างเดียว

```go
func TestCreateAccount(t *testing.T) {
    account, err := testQueries.CreateAccount(ctx, arg)

    if err != nil {
        t.Fatalf("CreateAccount returned error: %v", err)
    }

    if account.Owner != arg.Owner {
        t.Errorf("expected owner %q, got %q", arg.Owner, account.Owner)
    }

    if account.ID == 0 {
        t.Error("expected account ID to be generated")
    }
}
```

- `t.Error` และ `t.Errorf` ทำให้ test ล้มเหลว แต่ยังทำบรรทัดถัดไป
- `t.Fatal` และ `t.Fatalf` ทำให้ test ล้มเหลวและหยุด test function ทันที

### ใช้ `testify/require`

```go
import "github.com/stretchr/testify/require"

func TestCreateAccount(t *testing.T) {
    account, err := testQueries.CreateAccount(ctx, arg)

    require.NoError(t, err)
    require.Equal(t, arg.Owner, account.Owner)
    require.Equal(t, arg.Balance, account.Balance)
    require.NotZero(t, account.ID)
}
```

หาก assertion ของ `require` ไม่ผ่าน จะหยุด test function ทันที จึงเหมาะกับเงื่อนไขที่ขั้นตอนถัดไปจำเป็นต้องพึ่งพา เช่นต้องตรวจว่าไม่มี error ก่อนตรวจข้อมูลใน `account`

### `require` เทียบกับ `assert`

```go
require.NoError(t, err)          // ไม่ผ่านแล้วหยุด test ทันที
assert.Equal(t, expected, value) // ไม่ผ่านแล้วยังตรวจบรรทัดถัดไป
```

| `require` | `assert` |
| --- | --- |
| ไม่ผ่านแล้วหยุด test function | ไม่ผ่านแล้วยังทำบรรทัดถัดไป |
| ทำงานคล้าย `t.Fatal` / `t.FailNow` | ทำงานคล้าย `t.Error` / `t.Fail` |
| เหมาะกับเงื่อนไขที่จำเป็นต่อขั้นตอนต่อไป | เหมาะกับการตรวจหลายค่าเพื่อดู failures ทั้งหมด |

`testify` ไม่ได้มาแทนที่ package `testing` แต่ทำงานร่วมกัน เพราะ test function ยังต้องรับ `*testing.T`:

```go
func TestSomething(t *testing.T) {
    require.Equal(t, expected, actual)
}
```

สรุป: `testing` เพียงอย่างเดียวก็เพียงพอ ส่วน `testify` เป็นเครื่องมือเสริมที่ช่วยลดโค้ด `if` และทำให้ assertions อ่านง่ายขึ้น สำหรับการเรียนรู้ควรเข้าใจ `testing` ก่อน แล้วเลือกใช้ `testify` เมื่อช่วยให้ tests กระชับขึ้น

## ทำไม Go เรียก `TestMain` ก่อน Test Cases

`TestMain` เป็น entry point พิเศษที่กำหนดโดย package `testing` ของ Go ไม่จำเป็นต้องเรียกจาก `TestCreateAccount` ด้วยตนเอง

เมื่อรัน:

```bash
go test -timeout 30s ./db/sqlc -run '^TestCreateAccount$'
```

Go จะรวบรวมไฟล์ที่ลงท้ายด้วย `_test.go` และอยู่ใน package เดียวกัน เช่น:

```text
db/sqlc/
├── main_test.go      # มี TestMain
└── account_test.go   # มี TestCreateAccount
```

หากทั้งสองไฟล์ประกาศ `package db` Go จะมองว่าเป็น test package เดียวกัน ตัวแปรระดับ package เช่น `testQueries` จึงถูกใช้งานร่วมกันได้

Go test runner จะค้นหาฟังก์ชันที่มีชื่อและ signature ตรงตามรูปแบบนี้:

```go
func TestMain(m *testing.M)
```

เมื่อพบ Go จะใช้ฟังก์ชันนี้เป็น entry point ของการทดสอบ package โดยแนวคิดของ test runner ที่ Go สร้างขึ้นมีลักษณะประมาณนี้:

```go
func main() {
    m := createTestingM(
        TestCreateAccount,
        // test cases อื่น ๆ
    )

    TestMain(m)
}
```

จากนั้น `TestMain` เป็นผู้เรียก `m.Run()` เพื่อเริ่ม test cases:

```go
func TestMain(m *testing.M) {
    // setup
    testQueries = New(conn)

    code := m.Run() // TestCreateAccount เริ่มทำงานจากตรงนี้

    // cleanup
    conn.Close()
    os.Exit(code)
}
```

ลำดับการทำงานจริงคือ:

```text
go test
   │
   ├── compile package และไฟล์ *_test.go
   ├── พบและเรียก TestMain(m) อัตโนมัติ
   ├── ทำ setup เช่นกำหนด testQueries
   ├── TestMain เรียก m.Run()
   ├── m.Run() ใช้ -run กรองและรัน TestCreateAccount
   ├── กลับมาทำ cleanup ใน TestMain
   └── os.Exit(code) ส่งผลลัพธ์กลับ terminal หรือ CI
```

ตัวเลือก `-run` กรองเฉพาะ test cases ที่ `m.Run()` จะรัน ไม่ได้กรอง `TestMain` ดังนั้น `TestMain` ยังถูกเรียกเสมอเมื่อ package ประกาศฟังก์ชันนี้

สามารถพิสูจน์ลำดับด้วยการพิมพ์ข้อความ:

```go
func TestMain(m *testing.M) {
    fmt.Println("1: before m.Run")
    code := m.Run()
    fmt.Println("3: after m.Run")
    os.Exit(code)
}

func TestCreateAccount(t *testing.T) {
    fmt.Println("2: TestCreateAccount")
}
```

รันด้วย:

```bash
go test -v ./db/sqlc -run '^TestCreateAccount$'
```

จะเห็นผลลัพธ์ตามลำดับประมาณนี้:

```text
1: before m.Run
=== RUN   TestCreateAccount
2: TestCreateAccount
--- PASS: TestCreateAccount
3: after m.Run
PASS
```

ลำดับนี้ไม่ได้ขึ้นอยู่กับชื่อไฟล์หรือตำแหน่งของฟังก์ชัน แต่เกิดจากกฎของ package `testing` ที่กำหนดให้ `TestMain` เป็น entry point พิเศษของการทดสอบทั้ง package

## `Int63n` และ `Int64N` สำหรับการสุ่มตัวเลข

ชื่อฟังก์ชันที่ใช้ขึ้นอยู่กับ package ที่ import

### `math/rand`

package รุ่นเดิมใช้ `Int63n`:

```go
import "math/rand"

value := rand.Int63n(10)
```

Signature ของฟังก์ชันคือ:

```go
func Int63n(n int64) int64
```

แม้ชื่อจะเป็น `Int63n` แต่ค่าที่คืนมายังมีชนิดเป็น `int64` สาเหตุที่ใช้ชื่อ 63 เพราะฟังก์ชันคืนค่าเฉพาะจำนวนที่ไม่ติดลบ จึงใช้ 63 บิตสำหรับเก็บค่า ส่วนอีกหนึ่งบิตของ `int64` ปกติใช้ระบุเครื่องหมายบวกหรือลบ

`rand.Int63n(10)` จะสุ่มค่าในช่วง:

```text
0 <= value < 10
```

ดังนั้นค่าที่เป็นไปได้คือ `0` ถึง `9` โดยไม่รวม `10`

### `math/rand/v2`

package รุ่นใหม่ใช้ `Int64N` โดยตัว `N` เป็นตัวพิมพ์ใหญ่:

```go
import "math/rand/v2"

value := rand.Int64N(10)
```

`Int64N(10)` คืนค่า `int64` ตั้งแต่ `0` ถึง `9` เช่นเดียวกัน แต่ใช้รูปแบบชื่อที่สอดคล้องกับชนิด `int64` มากกว่า

| Package | ฟังก์ชัน | ชนิดค่าที่คืน | ช่วงเมื่อส่ง `10` |
| --- | --- | --- | --- |
| `math/rand` | `Int63n(10)` | `int64` | `0` ถึง `9` |
| `math/rand/v2` | `Int64N(10)` | `int64` | `0` ถึง `9` |

### การสุ่มค่าตั้งแต่ `min` ถึง `max`

สำหรับ `math/rand`:

```go
func RandomInt(min, max int64) int64 {
    return min + rand.Int63n(max-min+1)
}
```

สำหรับ `math/rand/v2`:

```go
func RandomInt(min, max int64) int64 {
    return min + rand.Int64N(max-min+1)
}
```

ตัวอย่างเมื่อ `min = 5` และ `max = 10`:

```text
max - min + 1 = 6
สุ่มค่า 0 ถึง 5
บวก min คือ 5
ผลลัพธ์จึงเป็น 5 ถึง 10
```

ต้องมี `+1` เพราะ `Int63n` และ `Int64N` ไม่รวมขอบบน หากไม่มี `+1` จะสุ่มได้เพียง `5` ถึง `9`

ข้อควรระวังคือ `max` ต้องไม่น้อยกว่า `min` เพราะ argument ที่ส่งให้ฟังก์ชันสุ่มต้องมากกว่า `0` ไม่เช่นนั้นจะเกิด panic

สำหรับโปรเจกต์ที่ใช้ `math/rand` ให้ใช้ `Int63n` ส่วนโปรเจกต์ใหม่ที่เลือกใช้ `math/rand/v2` สามารถใช้ `Int64N` ได้ ทั้งสองฟังก์ชันไม่เหมาะกับข้อมูลด้านความปลอดภัย เช่น password หรือ token ซึ่งควรใช้ `crypto/rand` แทน

## รัน Tests ทั้งโปรเจกต์และวัด Coverage

ใช้คำสั่งต่อไปนี้เพื่อรัน tests ทุก package พร้อมแสดงรายละเอียดและวัด test coverage:

```bash
go test -v -cover ./...
```

ความหมายของแต่ละส่วน:

- `go test` compile และรัน tests
- `-v` หรือ verbose แสดงชื่อและผลของแต่ละ test
- `-cover` แสดงเปอร์เซ็นต์ของ statements ที่ถูก execute ระหว่างการทดสอบ
- `./...` เลือก package ในตำแหน่งปัจจุบันและทุก package ในโฟลเดอร์ย่อย

ตัวอย่างผลลัพธ์:

```text
=== RUN   TestCreateAccount
--- PASS: TestCreateAccount (0.01s)
PASS
coverage: 72.5% of statements
ok   github.com/MumAroi/go-simplebank/db/sqlc
```

ค่า coverage หมายถึงสัดส่วน statements ที่ tests รันผ่าน ไม่ได้ยืนยันว่าโปรแกรมถูกต้องตามเปอร์เซ็นต์นั้น

หาก package ไม่มีไฟล์ test จะแสดงประมาณนี้:

```text
? github.com/MumAroi/go-simplebank/somepackage [no test files]
```

### ความหมายของ `./...`

`./...` เป็น package pattern ที่คำสั่ง Go เข้าใจ ไม่ใช่ wildcard `*` ของ shell

- `.` หมายถึง package ใน directory ปัจจุบัน
- `...` หมายถึง package ในตำแหน่งนั้นและโฟลเดอร์ย่อยทั้งหมดแบบ recursive

เปรียบเทียบคำสั่ง:

```bash
go test .          # ทดสอบเฉพาะ package ใน directory ปัจจุบัน
go test ./db/sqlc  # ทดสอบเฉพาะ package db/sqlc
go test ./db/...   # ทดสอบทุก package ที่อยู่ใต้ db
go test ./...      # ทดสอบทุก package ใน module ปัจจุบัน
```

รูปแบบ `./...` ใช้กับคำสั่ง Go อื่นได้ด้วย:

```bash
go build ./...
go vet ./...
go list ./...
```

Go จะเลือกเฉพาะโฟลเดอร์ที่เป็น Go package เช่นโฟลเดอร์ที่มีไฟล์ `.go` ส่วน `db/migration` ที่มีเฉพาะไฟล์ `.sql` จะไม่ถูกนับเป็น Go package

### สร้างรายงาน Coverage

บันทึกข้อมูล coverage ลงไฟล์:

```bash
go test -coverprofile=coverage.out ./...
```

ดู coverage แยกตาม function:

```bash
go tool cover -func=coverage.out
```

เปิดรายงานแบบ HTML:

```bash
go tool cover -html=coverage.out
```

สำหรับ integration tests ที่เชื่อมต่อ PostgreSQL ต้องเปิดฐานข้อมูลและ apply migrations ให้เรียบร้อยก่อนรัน tests ไม่เช่นนั้น test อาจล้มเหลวจากการเชื่อมต่อหรือไม่พบตาราง

## Struct, Interface, Context และ Dependency Injection

แนวคิดทั้งสี่ส่วนนี้มักทำงานร่วมกันใน Go application:

```text
Context ──> ควบคุมอายุของงาน
                  │
                  ▼
Service ──> Interface ──> Implementation
  struct       contract       database/mock
                  ▲
                  │
          Dependency Injection
```

### Struct

`struct` ใช้รวมข้อมูลหลาย field เป็นชนิดข้อมูลใหม่ คล้าย object data หรือ class ที่ไม่มี inheritance ใน TypeScript:

```go
type Queries struct {
    db DBTX
}
```

`Queries` มี field ชื่อ `db` ชนิด `DBTX` และสามารถมี methods ได้ผ่าน method receiver:

```go
func (q *Queries) CreateAccount(ctx context.Context, arg CreateAccountParams) (Account, error) {
    // q คือ Queries object ที่เรียก method
    // q.db คือ database dependency ภายใน object
}
```

สร้าง struct โดยตรงได้ว่า:

```go
queries := &Queries{db: conn}
```

หรือสร้างผ่าน factory function:

```go
func New(db DBTX) *Queries {
    return &Queries{db: db}
}

queries := New(conn)
```

### Interface

`interface` กำหนดชุดความสามารถหรือ methods ที่ค่าชนิดหนึ่งต้องมี โดยไม่ได้กำหนด implementation:

```go
type DBTX interface {
    ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
    QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
    QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}
```

Go ใช้ implicit interface implementation หากชนิดใดมี methods ครบ ก็ถือว่า implement interface โดยอัตโนมัติ ไม่ต้องเขียน `implements`

ตัวอย่าง `*sql.DB` และ `*sql.Tx` มี database methods ครบ จึงส่งเป็น `DBTX` ได้:

```go
queriesFromDB := New(db) // db เป็น *sql.DB
queriesFromTx := New(tx) // tx เป็น *sql.Tx
```

สามารถตรวจสอบตอน compile ได้ว่า type implement interface หรือไม่:

```go
var _ DBTX = (*sql.DB)(nil)
var _ DBTX = (*sql.Tx)(nil)
```

Interface ช่วยให้โค้ดพึ่งพาความสามารถที่ต้องการ แทนการผูกกับ implementation ชนิดเดียว

### Dependency Injection

Dependency คือสิ่งที่ object ต้องใช้เพื่อทำงาน เช่น `Queries` ต้องใช้ database ส่วน Dependency Injection (DI) คือการสร้าง dependency จากภายนอกแล้วส่งเข้ามาให้ object

```go
conn, err := sql.Open("postgres", databaseURL)
if err != nil {
    log.Fatal(err)
}

queries := New(conn)
```

ในตัวอย่างนี้:

- `conn` คือ dependency
- `New(conn)` คือการ inject dependency
- `Queries` เก็บ dependency ไว้ใน field `db`

```text
สร้าง conn ภายนอก
       │
       ▼
   New(conn)
       │
       ▼
Queries{db: conn}
       │
       ▼
CreateAccount ใช้ q.db
```

การ inject dependency ดีกว่าการให้ `Queries` เปิด connection เอง เพราะผู้เรียกสามารถควบคุม configuration, lifecycle และเปลี่ยน implementation สำหรับ tests ได้

ตัวอย่าง service ที่รับ interface:

```go
type AccountStore interface {
    GetAccount(ctx context.Context, id int64) (Account, error)
}

type AccountService struct {
    store AccountStore
}

func NewAccountService(store AccountStore) *AccountService {
    return &AccountService{store: store}
}
```

ตอนใช้งานจริงสามารถ inject PostgreSQL store:

```go
service := NewAccountService(postgresStore)
```

ตอน unit test สามารถ inject fake หรือ mock โดยไม่ต้องเปิดฐานข้อมูล:

```go
type FakeStore struct {
    account Account
}

func (f *FakeStore) GetAccount(ctx context.Context, id int64) (Account, error) {
    return f.account, nil
}

fake := &FakeStore{account: Account{ID: 1, Owner: "Tom"}}
service := NewAccountService(fake)
```

DI ไม่ได้บังคับว่าต้องใช้ interface แต่ interface ทำให้สลับ implementation และเขียน unit tests ได้ง่ายขึ้น

### Context

`context.Context` ใช้ควบคุมอายุของงานและส่งสัญญาณตามลำดับการเรียกฟังก์ชัน เช่น:

- ยกเลิกงาน
- กำหนด timeout
- กำหนด deadline
- ส่งข้อมูลที่ผูกกับ request เช่น request ID

Context มักถูกส่งเป็น parameter ตัวแรก:

```go
func (s *AccountService) GetAccount(ctx context.Context, id int64) (Account, error) {
    return s.store.GetAccount(ctx, id)
}
```

Context เดียวกันจะถูกส่งต่อผ่านแต่ละชั้น:

```text
HTTP Request
    │ r.Context()
    ▼
Handler
    │ ctx
    ▼
Service
    │ ctx
    ▼
Store / Database Query
```

หาก client ยกเลิก HTTP request ตัว `r.Context()` จะถูกยกเลิก และ database driver สามารถหยุดรอ query ที่ใช้ context เดียวกันได้

#### Root Context

ใช้ `context.Background()` เป็น context เริ่มต้นใน `main`, initialization หรือ tests:

```go
account, err := testQueries.GetAccount(context.Background(), accountID)
```

`Background` ไม่มี timeout, deadline หรือการยกเลิกอัตโนมัติ ส่วน `context.TODO()` ใช้เป็น placeholder เมื่อยังไม่แน่ใจว่าควรรับ context จากที่ใด

#### Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

account, err := queries.GetAccount(ctx, accountID)
```

หาก query ใช้เวลาเกิน 2 วินาที context จะหมดเวลาและคืน error เช่น `context deadline exceeded` ควรเรียก `cancel()` เสมอเพื่อคืนทรัพยากร แม้งานจะเสร็จก่อน timeout

#### Cancellation

```go
ctx, cancel := context.WithCancel(context.Background())

go doWork(ctx)
cancel()
```

ภายในงานสามารถรอสัญญาณยกเลิกผ่าน `ctx.Done()` และตรวจสาเหตุผ่าน `ctx.Err()`

#### แนวทางการใช้ Context

- รับ context เป็น parameter ตัวแรก เช่น `func DoSomething(ctx context.Context, ...)`
- ส่ง context เดิมต่อไปยังชั้นล่าง
- ใน HTTP handler ให้ใช้ `r.Context()` แทนการสร้าง `Background()` ใหม่
- อย่าส่ง `nil`; หากไม่มี context ให้ใช้ `context.Background()`
- ไม่ควรใช้ context เก็บ database connection, config หรือ optional parameters
- ใช้ `WithValue` เฉพาะข้อมูลที่ผูกกับ request เช่น request ID หรือ trace ID

### เปรียบเทียบกับ TypeScript

Go:

```go
type AccountStore interface {
    GetAccount(ctx context.Context, id int64) (Account, error)
}

type AccountService struct {
    store AccountStore
}

func NewAccountService(store AccountStore) *AccountService {
    return &AccountService{store: store}
}
```

TypeScript:

```ts
interface AccountStore {
  getAccount(id: number, signal?: AbortSignal): Promise<Account>;
}

class AccountService {
  constructor(private store: AccountStore) {}
}
```

| Go | TypeScript |
| --- | --- |
| `struct` | `class` หรือ object shape |
| `interface` | `interface` |
| `NewAccountService(store)` | `new AccountService(store)` |
| `q *Queries` | `this` |
| `context.Context` | ใกล้เคียง `AbortSignal` สำหรับ cancellation |
| `(value, error)` | `Promise<value>` และ exception/error |

สรุป: `struct` เก็บ state และ dependencies, `interface` กำหนดความสามารถ, Dependency Injection ส่ง implementation เข้ามาจากภายนอก และ `context` ควบคุมอายุของงานที่ไหลผ่านแต่ละชั้นของ application

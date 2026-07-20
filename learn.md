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

## Database Transaction และ ACID

Transaction คือการรวม SQL operations หลายรายการให้เป็นงานหนึ่งชุด โดยต้องสำเร็จทั้งหมดหรือยกเลิกทั้งหมด

ตัวอย่างการโอนเงินจากบัญชี A ไปบัญชี B ประกอบด้วยหลาย operation:

```text
1. สร้างรายการโอน
2. หักเงินจากบัญชี A
3. เพิ่มเงินให้บัญชี B
4. บันทึกรายการเงินออกและเงินเข้า
```

ถ้าหักเงินจาก A สำเร็จ แต่เพิ่มเงินให้ B ล้มเหลว ระบบต้องย้อนการเปลี่ยนแปลงทั้งหมด ไม่เช่นนั้นข้อมูลจะอยู่ในสถานะไม่สมบูรณ์

รูปแบบ SQL พื้นฐาน:

```sql
BEGIN;

UPDATE accounts
SET balance = balance - 100
WHERE id = 1;

UPDATE accounts
SET balance = balance + 100
WHERE id = 2;

COMMIT;
```

- `BEGIN` เริ่ม transaction
- `COMMIT` ยืนยันการเปลี่ยนแปลงทั้งหมด
- `ROLLBACK` ย้อนการเปลี่ยนแปลงทั้งหมดใน transaction

หาก operation ใดล้มเหลว application ควรสั่ง:

```sql
ROLLBACK;
```

### ACID Properties

ACID คือคุณสมบัติสี่ข้อที่ช่วยให้ transaction เชื่อถือได้

#### Atomicity — สำเร็จทั้งหมดหรือไม่เกิดขึ้นเลย

ทุก operation ใน transaction ถูกมองเป็นหน่วยเดียว:

```text
หัก A สำเร็จ + เพิ่ม B สำเร็จ  ──> COMMIT
หัก A สำเร็จ + เพิ่ม B ล้มเหลว ──> ROLLBACK ทั้งหมด
```

จึงไม่ควรเกิดสถานะที่บัญชีต้นทางถูกหักเงิน แต่บัญชีปลายทางไม่ได้รับเงิน

#### Consistency — ข้อมูลยังรักษากฎของระบบ

ก่อนและหลัง transaction ข้อมูลต้องอยู่ในสถานะที่ถูกต้องตาม database constraints และ business rules ที่ application บังคับใช้ เช่น:

- Primary Key ต้องไม่ซ้ำ
- Foreign Key ต้องอ้างถึงข้อมูลที่มีอยู่
- ค่า currency ต้องอยู่ในชุดที่ระบบรองรับ
- เงินรวมก่อนและหลังการโอนควรเท่าเดิม
- หากไม่อนุญาตยอดติดลบ balance ต้องไม่น้อยกว่า `0`

กฎบางส่วนควรบังคับด้วย constraint ในฐานข้อมูล:

```sql
balance BIGINT NOT NULL CHECK (balance >= 0)
```

ฐานข้อมูลรับประกัน constraints ที่ประกาศไว้ แต่ business rules ที่ไม่ได้เขียนเป็น constraint ยังเป็นความรับผิดชอบของ application

#### Isolation — Transactions ที่ทำพร้อมกันไม่รบกวนกันอย่างผิดพลาด

หากสอง transactions อ่านและแก้ไขบัญชีเดียวกันพร้อมกัน อาจเกิดปัญหา เช่น lost update หรือใช้เงินเกินยอด ฐานข้อมูลจึงใช้ snapshot, isolation level และ locking เพื่อควบคุม concurrency

ตัวอย่างล็อกแถวก่อนแก้ไข:

```sql
SELECT *
FROM accounts
WHERE id = $1
FOR UPDATE;
```

Isolation levels หลักของ PostgreSQL:

- `Read Committed` เป็นค่าเริ่มต้น แต่ละ statement เห็นเฉพาะข้อมูลที่ commit แล้วก่อน statement นั้นเริ่ม
- `Repeatable Read` transaction อ่านจาก snapshot เดิมตลอด transaction
- `Serializable` เข้มงวดที่สุด ให้ผลเสมือน transactions ทำทีละรายการ แต่อาจเกิด serialization failure และ application ต้อง retry

Isolation level ที่เข้มงวดขึ้นช่วยลด concurrency anomalies แต่มีต้นทุนด้าน locking, retry หรือ throughput จึงควรเลือกตามความต้องการของงาน

#### Durability — Commit แล้วข้อมูลต้องคงอยู่

เมื่อฐานข้อมูลยืนยันว่า `COMMIT` สำเร็จ ข้อมูลต้องคงอยู่แม้ server crash หรือ restart PostgreSQL ใช้กลไก Write-Ahead Log (WAL) เพื่อช่วยกู้คืนข้อมูลที่ commit แล้ว

### Transaction ใน Go

รูปแบบพื้นฐานด้วย `database/sql`:

```go
func execTx(
    ctx context.Context,
    db *sql.DB,
    fn func(*Queries) error,
) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }

    queries := New(tx)

    if err := fn(queries); err != nil {
        if rollbackErr := tx.Rollback(); rollbackErr != nil {
            return fmt.Errorf(
                "transaction error: %v, rollback error: %v",
                err,
                rollbackErr,
            )
        }
        return err
    }

    return tx.Commit()
}
```

ลำดับการทำงาน:

```text
BeginTx
   │
   ▼
ทำ operations ผ่าน Queries
   │
   ├── มี error ──> Rollback
   │
   └── สำเร็จ ────> Commit
```

`New(tx)` ใช้งานได้เพราะ `*sql.Tx` มี methods ครบตาม interface `DBTX` เช่นเดียวกับ `*sql.DB` generated queries จึงทำงานได้ทั้งกับ connection pool ปกติและ transaction

#### การทำงานของ `Store.execTx`

ตัวอย่าง implementation ใน `Store`:

```go
func (store *Store) execTx(
    ctx context.Context,
    fn func(*Queries) error,
) error {
    tx, err := store.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }

    q := New(tx)
    err = fn(q)
    if err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("tx err: %w, rb err: %w", err, rbErr)
        }
        return err
    }

    return tx.Commit()
}
```

`fn func(*Queries) error` คือ callback ที่ผู้เรียกส่งเข้ามาเพื่อระบุ operations ที่ต้องทำภายใน transaction:

```go
err := store.execTx(ctx, func(q *Queries) error {
    _, err := q.CreateTransfer(ctx, arg)
    return err
})
```

เมื่อ `execTx` เรียก `fn(q)` anonymous function ด้านบนจะทำงานโดยใช้ `q` ที่ผูกกับ `tx`

หน้าที่ของแต่ละขั้นตอน:

1. `store.db.BeginTx(ctx, nil)` เริ่ม transaction โดย `nil` หมายถึงใช้ transaction options ค่าเริ่มต้น
2. `q := New(tx)` สร้าง `Queries` ที่ส่งทุก query ผ่าน transaction เดียวกัน
3. `fn(q)` รันชุด operations ที่ผู้เรียกกำหนด
4. หาก `fn` คืน error ให้ `Rollback` และคืน error กลับ
5. หาก `Rollback` ล้มเหลวด้วย ให้รวม transaction error และ rollback error
6. หาก `fn` สำเร็จ ให้ `Commit` และคืน commit error หากมี

ภายใน callback ต้องเรียก query ผ่าน `q`:

```go
q.CreateTransfer(ctx, arg) // อยู่ใน transaction
```

ไม่ควรเรียกผ่าน `store`:

```go
store.CreateTransfer(ctx, arg) // ใช้ connection pool ปกติ อาจอยู่นอก transaction
```

ดังนั้น `execTx` แยกความรับผิดชอบสองส่วนออกจากกัน: helper จัดการ lifecycle ของ transaction ส่วน callback ระบุ business operations ที่ต้องทำเป็นชุดเดียวกัน

ข้อควรระวัง:

- ตรวจ error จาก `BeginTx`, operations, `Rollback` และ `Commit`
- ส่ง `context.Context` เพื่อให้ transaction ถูกยกเลิกได้เมื่อ request หมดเวลาหรือถูกยกเลิก
- อย่าทำงานที่ใช้เวลานานหรือเรียก external service ภายใน transaction โดยไม่จำเป็น เพราะจะถือ connection และ locks นานขึ้น
- เมื่อหลาย transactions ล็อกหลายแถว ควรล็อกตามลำดับที่แน่นอนเพื่อลดโอกาสเกิด deadlock
- transaction ที่ระดับ `Serializable` อาจต้องมี retry เมื่อเกิด serialization failure

สรุป: Transaction รวมหลาย operations เป็นงานหนึ่งชุด ส่วน ACID อธิบายคุณสมบัติที่ทำให้งานชุดนั้นเชื่อถือได้ ได้แก่ Atomicity, Consistency, Isolation และ Durability

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

- `false` เริ่ม application ง่าย ไม่ต้อง prepare และปิด statements เหมาะเมื่อ query ยังไม่มากหรือยังไม่พบปัญหาด้าน performance
- `true` อาจช่วย query เดิมที่ถูกเรียกซ้ำจำนวนมาก และทำให้พบข้อผิดพลาดตอน prepare ตั้งแต่เริ่ม application
- `true` มีต้นทุนตอนเริ่มระบบ ใช้ resource เพิ่ม และต้องจัดการ lifecycle เช่นเรียก `Close()`
- prepared statements ไม่ได้รับประกันว่าจะเร็วกว่าเสมอ ควร benchmark ด้วย workload จริง

โปรเจกต์นี้ใช้ `database/sql` และมี query ไม่มาก จึงคง `false` ไว้ก่อนได้ หากเปลี่ยนไปใช้ `pgx/v5` ตัว driver มี implicit prepared statement support อยู่แล้ว

#### `emit_interface: false`

ไม่สร้าง interface ที่รวม query methods หากตั้งเป็น `true` จะสร้าง interface เช่น `Querier` ซึ่งช่วยในการทำ mock, dependency injection และ unit test

```go
type Querier interface {
    GetAccount(ctx context.Context, id int64) (Account, error)
    ListAccounts(ctx context.Context) ([]Account, error)
}
```

ข้อดีเมื่อเปิดเป็น `true`:

- ใช้ Dependency Injection ได้สะดวก
- ส่ง mock หรือ fake เข้าไปทดสอบ business logic โดยไม่ต้องต่อฐานข้อมูลจริงได้
- compiler ช่วยตรวจว่า implementation มี methods ครบ

ข้อควรพิจารณา:

- `Querier` จะรวม query methods ทั้งหมดและอาจใหญ่เกินความจำเป็นเมื่อระบบโตขึ้น
- mock ที่ implement interface ทั้งก้อนต้องปรับตามเมื่อมี method ใหม่
- transaction ที่รวมหลาย query อาจยังต้องมี `Store` interface ของ application เอง

ถ้าแต่ละ service ใช้เพียงไม่กี่ query สามารถประกาศ interface ขนาดเล็กใกล้ผู้ใช้งานแทนได้:

```go
type AccountStore interface {
    GetAccount(ctx context.Context, id int64) (Account, error)
    CreateAccount(ctx context.Context, arg CreateAccountParams) (Account, error)
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

## พื้นฐานภาษา Go ที่ใช้ในโปรเจกต์

### Boilerplate คืออะไร

Boilerplate คือโค้ดรูปแบบเดิมที่จำเป็นต้องเขียนซ้ำเพื่อให้ระบบทำงาน แต่ไม่ใช่ business logic หลัก เช่นการเรียก query, สร้างตัวแปรรับผลลัพธ์, `Scan` fields และส่งต่อ error

`sqlc` ช่วยสร้าง boilerplate ของ database layer จาก SQL ทำให้เราเรียก method ที่ type-safe ได้โดยไม่ต้องเขียนโค้ดเดิมซ้ำทุก query

### `int` และ `int64`

- `int` มีขนาดตามสถาปัตยกรรมของเครื่อง เหมาะกับ index, `len()`, `cap()` และ loop
- `int64` มีขนาด 64 บิตเสมอ เหมาะกับ PostgreSQL `BIGINT`, ID, balance และข้อมูลที่ต้องมีขนาดแน่นอน
- แม้บนเครื่อง 64-bit ทั้งสองอาจมีขนาดเท่ากัน แต่ Go ถือว่าเป็นคนละ type

```go
var count int = 10
var balance int64 = 100

balance = int64(count) // ต้องแปลง type อย่างชัดเจน
```

การแปลง `int64` เป็น `int` ต้องระวังค่าสูงเกินช่วงของ `int` โดยเฉพาะบนระบบ 32-bit

### Pointer

Pointer คือตัวแปรที่เก็บที่อยู่ของตัวแปรอีกตัว:

```go
x := 10
p := &x  // p มี type เป็น *int และเก็บที่อยู่ของ x
*p = 20 // dereference แล้วแก้ x ผ่าน pointer
```

- `&x` ขอที่อยู่ของ `x`
- `*int` หมายถึง type ที่เป็น pointer ไปยัง `int`
- `*p` เข้าถึงค่าที่ `p` ชี้อยู่
- pointer มีค่าเป็น `nil` ได้ และการ dereference `nil` จะทำให้ panic

Go ส่ง argument แบบ copy เสมอ การส่ง pointer คือการ copy ที่อยู่ ทำให้ function เข้าถึงและแก้ข้อมูลต้นฉบับได้:

```go
func increase(n *int) {
    (*n)++
}

x := 10
increase(&x) // x กลายเป็น 11
```

Pointer receiver เช่น `func (q *Queries) ...` เหมาะเมื่อ method ต้องแก้ struct เดิม หลีกเลี่ยงการ copy struct หรือรักษาความสม่ำเสมอของ method set

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

## การสุ่มตัวเลขด้วย `math/rand`

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

สำหรับโปรเจกต์ใหม่ที่ใช้ Go 1.22 ขึ้นไป แนะนำ `math/rand/v2` ส่วน `math/rand` เหมาะกับการดูแลโค้ดเดิม ทั้งสอง package เป็น pseudo-random และไม่เหมาะกับ password, token หรือ secret ซึ่งควรใช้ `crypto/rand`

`math/rand/v2` ไม่มี `Intn()` แบบ package เดิม แต่ใช้ `IntN()` โดย `N` เป็นตัวพิมพ์ใหญ่:

```go
index := rand.IntN(10)     // รับและคืนค่า int
amount := rand.Int64N(10) // รับและคืนค่า int64
```

- `IntN` เหมาะกับ index และค่าจาก `len()` ซึ่งเป็น `int`
- `Int64N` เหมาะกับ ID, balance หรือ PostgreSQL `BIGINT` ซึ่งเป็น `int64`
- ทั้งคู่สุ่มในช่วง `[0, n)` และ panic เมื่อ `n <= 0`

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

หาก `min = 1` และ `max = 10`:

```text
max - min + 1 = 10
rand.Int64N(10) สุ่มได้ 0 ถึง 9
นำผลที่สุ่มได้ไปบวก min จึงได้ 1 ถึง 10
```

เลข `10` ที่ส่งเข้า `Int64N` คือจำนวนค่าที่เป็นไปได้ ไม่ใช่ผลลัพธ์ที่จะนำไปบวกกับ `min`

สูตรเดียวกันรองรับช่วงติดลบ เช่น `min = -5`, `max = 5` จะสุ่มได้ตั้งแต่ `-5` ถึง `5` รวมขอบทั้งสองด้าน หากต้องการค่าติดลบเสมอตั้งแต่ `-1` ถึง `-10` ใช้:

```go
n := -(rand.IntN(10) + 1)
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

`PASS` กับ coverage วัดคนละเรื่อง:

- `PASS` หมายถึง test cases ที่รันไม่มี assertion ล้มเหลวหรือ panic
- coverage `72.5%` หมายถึง test วิ่งผ่านประมาณ 72.5% ของ statements ที่นำมาวัด
- methods หรือ branches ที่ไม่มี test เรียกยังลด coverage ได้ แม้ tests ที่มีอยู่จะผ่านทั้งหมด

ดังนั้น coverage สูงไม่ได้รับประกันว่า test ดี เพราะ test อาจเรียก statement โดยไม่ได้ตรวจผลลัพธ์สำคัญ ควรมี assertions ตรวจทั้ง error และค่าที่คืนมา

### Unit Test เทียบกับ Integration Test

Unit test และ integration test ตรวจคนละส่วนของระบบ จึงควรเลือก implementation ให้ตรงกับสิ่งที่ต้องการทดสอบ

#### Unit Test ใช้ Fake หรือ Mock

Unit test ของ service ควรทดสอบ business logic โดยแยกออกจาก PostgreSQL จึง inject fake หรือ mock ที่ implement interface เดียวกับ store จริง:

```go
fake := &FakeStore{
    account: Account{ID: 1, Owner: "Tom"},
}

service := NewAccountService(fake)
account, err := service.GetAccount(context.Background(), 1)
```

การทดสอบนี้เหมาะสำหรับตรวจว่า service ส่ง parameter ถูกต้อง จัดการผลลัพธ์หรือ error ถูกต้อง และบังคับ business rules ถูกต้อง โดยไม่ตรวจ SQL หรือการเชื่อมต่อฐานข้อมูล

#### Integration Test ใช้ Database จริง

หากต้องการตรวจว่า SQL, parameter, การ `Scan` และ PostgreSQL ทำงานร่วมกันถูกต้อง ให้เรียก generated query จริงกับ test database:

```go
func TestGetAccount(t *testing.T) {
    created := createRandomAccount(t)

    account, err := testQueries.GetAccount(
        context.Background(),
        created.ID,
    )

    require.NoError(t, err)
    require.Equal(t, created.ID, account.ID)
    require.Equal(t, created.Owner, account.Owner)
}
```

รูปแบบนี้เป็น integration test เพราะทดสอบการทำงานร่วมกันหลายส่วน ต้องเปิด PostgreSQL, apply migrations และจัดการ test data จึงช้ากว่าและพึ่งพา environment มากกว่า unit test

```text
AccountService ── unit test ด้วย FakeStore
       │
       ▼
AccountStore interface
       │
       ▼
sqlc Queries ─── integration test ด้วย PostgreSQL จริง
```

ถ้า service แค่ส่งต่อไปยัง store โดยไม่มี business logic unit test อาจให้ประโยชน์ไม่มาก แต่เมื่อมี validation, authorization หรือการแปลง error ควรมี unit tests สำหรับแต่ละกรณี ส่วน database query ควรมี integration test แยกต่างหาก ระบบหนึ่งจึงสามารถและมักควรมีทั้งสองแบบ

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
go test -coverprofile coverage.out ./...
```

ดู coverage แยกตาม function:

```bash
go tool cover -func coverage.out
```

เปิดรายงานแบบ HTML:

```bash
go tool cover -html coverage.out
```

รายงานแบบ function ช่วยระบุว่าฟังก์ชันใดยังมี statements ที่ tests ไม่ได้รันผ่าน:

```text
account.sql.go:30:  CreateAccount  100.0%
account.sql.go:82:  ListAccounts   73.3%
db.go:27:           WithTx         0.0%
random.go:10:       RandomInt      0.0%
total:                            60.5%
```

- `100.0%` หมายถึงทุก statement ในฟังก์ชันนั้นถูกรันผ่าน
- `0.0%` หมายถึงยังไม่มี test รันผ่านฟังก์ชันนั้น
- ค่าระหว่างกลาง เช่น `73.3%` หมายถึงทดสอบเส้นทางปกติแล้ว แต่อาจยังไม่ผ่านบาง branch หรือ error path
- `total` คือ coverage รวมของ packages ที่อยู่ใน profile

ข้อความจาก `go test` เช่น:

```text
ok   github.com/MumAroi/go-simplebank/db/sqlc  coverage: 83.9% of statements
```

เป็น coverage ของ package `db/sqlc` เท่านั้น จึงอาจไม่เท่ากับ `total` หาก package อื่น เช่น `util` ยังมี coverage `0%`

### ความหมายของ `covered` และ `not covered`

เมื่อเปิดรายงานด้วย:

```bash
go tool cover -html coverage.out
```

รายงานจะแสดงว่า statements ส่วนใดถูก tests รันผ่าน:

- `covered` หมายถึงมี test อย่างน้อยหนึ่งกรณี execute statement นั้น
- `not covered` หมายถึงยังไม่มี test ใด execute statement นั้น
- สีเขียวโดยทั่วไปหมายถึง `covered`
- สีแดงโดยทั่วไปหมายถึง `not covered`
- สีเทามักเป็นส่วนที่ไม่ใช่ executable statement เช่น comment, type declaration หรือวงเล็บ

ตัวอย่าง:

```go
func Divide(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("cannot divide by zero")
    }

    return a / b, nil
}
```

หาก test เรียกเฉพาะ `Divide(10, 2)` บรรทัด `return a / b, nil` จะเป็น `covered` แต่ error path ภายใน `b == 0` จะเป็น `not covered` เพราะยังไม่มี test ส่งค่า `b` เป็น `0`

สามารถเพิ่ม test สำหรับเส้นทางนั้นได้ว่า:

```go
func TestDivideByZero(t *testing.T) {
    result, err := Divide(10, 0)

    require.Error(t, err)
    require.Zero(t, result)
}
```

`not covered` ไม่ได้หมายความว่าโค้ดผิด แต่หมายความว่า test suite ปัจจุบันยังไม่ได้ทำให้โปรแกรมเดินผ่านส่วนนั้น อาจเป็นฟังก์ชันที่ยังไม่มี test, อีกฝั่งของ `if`, กรณี `default` ใน `switch`, error path หรือ generated code ที่จำลองข้อผิดพลาดได้ยาก

รายงาน HTML ช่วยหาเป็นรายบรรทัด โดยทั่วไปสีเขียวคือ statement ที่ tests รันผ่าน และสีแดงคือ statement ที่ยังไม่ถูกรัน วิธีนี้เหมาะที่สุดสำหรับค้นหาว่าต้องเพิ่ม test ตรงไหน

### วิธีเพิ่ม Coverage

1. เริ่มจากฟังก์ชันที่เป็น `0%` และเพิ่ม test สำหรับพฤติกรรมหลัก
2. เพิ่มกรณี input ที่ขอบเขต เช่น ค่าว่าง ค่าต่ำสุด ค่าสูงสุด หรือไม่พบข้อมูล
3. เพิ่มกรณี error สำหรับ code paths ที่จัดการข้อผิดพลาด
4. ตรวจค่าที่คืนและ error ด้วย assertions ไม่ใช่เพียงเรียกฟังก์ชันเพื่อเพิ่มตัวเลข
5. รัน profile ใหม่และตรวจด้วย `go tool cover -func` หรือรายงาน HTML

ตัวอย่าง test สำหรับ utility function:

```go
func TestRandomInt(t *testing.T) {
    for range 100 {
        value := RandomInt(10, 20)
        require.GreaterOrEqual(t, value, int64(10))
        require.LessOrEqual(t, value, int64(20))
    }
}

func TestRandomString(t *testing.T) {
    value := RandomString(10)
    require.Len(t, value, 10)
    require.NotEmpty(t, value)
}
```

บาง error paths ใน generated code เช่น error จาก `rows.Scan()` หรือ `rows.Err()` อาจทดสอบยากด้วยฐานข้อมูลจริงและต้องใช้ mock database การบังคับให้ generated code เป็น `100%` จึงอาจเพิ่มความซับซ้อนโดยไม่ได้เพิ่มความมั่นใจมากนัก

เป้าหมายที่สำคัญกว่า coverage `100%` คือ business logic, กรณีสำคัญ และ error handling ที่มีโอกาสเกิดจริงได้รับการทดสอบ พร้อม assertions ที่ตรวจพฤติกรรมอย่างถูกต้อง

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

#### สร้าง Struct โดยตรงเทียบกับใช้ `New`

สองรูปแบบนี้ให้ผลลัพธ์พื้นฐานเหมือนกัน คือได้ `*Queries` ที่มี `conn` เก็บอยู่ใน field `db`:

```go
// สร้างด้วย struct literal โดยตรง
queries := &Queries{db: conn}

// สร้างผ่าน constructor function
func New(db DBTX) *Queries {
    return &Queries{db: db}
}

queries := New(conn)
```

การสร้างโดยตรงมีข้อดีคือสั้นและมองเห็นทันทีว่ากำหนด field ใด เหมาะกับ struct ธรรมดาที่ไม่มี validation หรือขั้นตอน setup และใช้งานภายใน package เดียวกัน แต่ผู้เรียกต้องรู้โครงสร้างภายใน และถ้าวิธีสร้างเปลี่ยนก็อาจต้องแก้ทุกจุดที่สร้าง struct

การสร้างผ่าน `New` มีข้อดีดังนี้:

- ซ่อนรายละเอียดภายใน ผู้เรียกไม่ต้องรู้ว่า dependency ถูกเก็บใน field ชื่ออะไร
- รวม validation, default values และขั้นตอน setup ไว้ในที่เดียว
- เปลี่ยนโครงสร้างภายในภายหลังได้โดยไม่ต้องแก้โค้ดผู้เรียก
- package อื่นเรียกใช้ได้ แม้ struct จะมี unexported fields

ข้อเสียคือเพิ่ม function อีกชั้น และอาจดูเกินความจำเป็นสำหรับ struct ข้อมูลธรรมดาที่ไม่มีข้อกำหนดในการสร้าง

ในกรณี `Queries` field `db` ขึ้นต้นด้วยตัวพิมพ์เล็ก จึงเป็น unexported และเข้าถึงได้เฉพาะ package เดียวกัน โค้ดใน package อื่นจึงทำแบบนี้ไม่ได้:

```go
queries := &db.Queries{db: conn} // compile error: เข้าถึง db ไม่ได้
```

แต่สามารถใช้ API ที่ `sqlc` เตรียมไว้ได้:

```go
queries := db.New(conn)
```

ดังนั้น `Queries` ควรสร้างผ่าน `New` ส่วน struct ที่เป็นข้อมูลธรรมดาและเปิด fields ให้กำหนดโดยตั้งใจ สามารถใช้ struct literal โดยตรงได้

#### Embedded Field

Embedded field คือการใส่ type เข้าไปใน `struct` โดยไม่ตั้งชื่อ field เอง:

```go
type Store struct {
    *Queries
    db *sql.DB
}
```

ในตัวอย่างนี้ `*Queries` เป็น embedded field ส่วน `db *sql.DB` เป็น field ปกติที่มีชื่อ `db`

เปรียบเทียบกับ field ปกติ:

```go
type Store struct {
    queries *Queries
}

store.queries.CreateAccount(ctx, arg)
```

เมื่อใช้ embedded field Go จะใช้ชื่อ type เป็นชื่อ field จึงเข้าถึงได้ว่า:

```go
store.Queries.CreateAccount(ctx, arg)
```

และ methods ของ `Queries` จะถูก promote ขึ้นมาให้เรียกผ่าน `Store` โดยตรงได้:

```go
store.CreateAccount(ctx, arg)
```

การเรียกสองรูปแบบนี้ทำงานผ่าน `Queries` ตัวเดียวกัน:

```go
store.Queries.CreateAccount(ctx, arg)
store.CreateAccount(ctx, arg)
```

การที่ method ของ embedded type เรียกผ่าน struct ภายนอกได้โดยตรงเรียกว่า **method promotion**

สร้าง `Store` และกำหนด embedded field ได้ว่า:

```go
func NewStore(db *sql.DB) *Store {
    return &Store{
        db:      db,
        Queries: New(db),
    }
}
```

แม้ประกาศเป็น pointer `*Queries` ชื่อ field ที่ใช้ใน struct literal ยังคงเป็น `Queries`

Embedding เป็น composition หรือความสัมพันธ์แบบ “มี”:

```text
Store has a Queries
```

ไม่ใช่ inheritance หรือความสัมพันธ์แบบ “เป็น” ดังนั้น `*Store` ไม่ได้กลายเป็น `*Queries` และไม่สามารถส่ง `store` ให้ฟังก์ชันที่รับ `*Queries` โดยตรง:

```go
func RunQueries(q *Queries) {}

RunQueries(store)         // compile error
RunQueries(store.Queries) // ถูกต้อง
```

หาก `Store` และ embedded `Queries` มี method ชื่อเดียวกัน Go จะเลือก method ของ `Store` ก่อน:

```go
store.Save()         // เรียก Store.Save
store.Queries.Save() // เรียก Queries.Save โดยตรง
```

ข้อควรระวังคือ embedded pointer อาจเป็น `nil`:

```go
store := &Store{}
store.CreateAccount(ctx, arg) // อาจ panic เพราะ store.Queries เป็น nil
```

จึงควรสร้างผ่าน `NewStore(db)` เพื่อกำหนดทั้ง `db` และ embedded `Queries` ให้พร้อมใช้งาน

ในโปรเจกต์นี้ `Store` embed `*Queries` เพื่อเรียก generated query methods ได้โดยตรง และเก็บ `db *sql.DB` แยกไว้เพื่อเริ่ม transaction ด้วย `BeginTx`

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

คำว่า implement หมายถึงการทำตามข้อกำหนดของ interface ครบทั้งชื่อ method, parameters และ return values ตัวอย่าง:

```go
type Speaker interface {
    Speak() string
}

type Dog struct{}

func (Dog) Speak() string {
    return "Woof!"
}

var speaker Speaker = Dog{} // Dog implement Speaker โดยอัตโนมัติ
```

ถ้า method ขาดหรือ signature ไม่ตรงจะ compile ไม่ผ่าน นี่เรียกว่า implicit เพราะ `Dog` ไม่ต้องประกาศ `implements Speaker` เอง

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

func (s *AccountService) GetAccount(ctx context.Context, id int64) (Account, error) {
    return s.store.GetAccount(ctx, id)
}
```

Interface ถูกใช้งานตรง type ของ field และ parameter คือ `store AccountStore` ส่วนจุดที่เรียก method ผ่าน interface คือ `s.store.GetAccount(ctx, id)` Logic ภายใน store แต่ละตัวไม่จำเป็นต้องเหมือนกัน ขอเพียง method signature ตรงตาม contract ของ `AccountStore`

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

`*FakeStore` อ่านค่าที่เตรียมไว้ใน memory ส่วน PostgreSQL store อ่านจากฐานข้อมูลจริง แม้ logic ภายในต่างกัน ทั้งคู่ใช้เป็น `AccountStore` ได้เพราะมี `GetAccount(context.Context, int64) (Account, error)` ตรงกัน

DI ไม่ได้บังคับว่าต้องใช้ interface แต่ interface ทำให้สลับ implementation และเขียน unit tests ได้ง่ายขึ้น

### Context

`context.Context` ใช้ควบคุมอายุของงานและส่งสัญญาณตามลำดับการเรียกฟังก์ชัน เช่น:

- ยกเลิกงาน
- กำหนด timeout
- กำหนด deadline
- ส่งข้อมูลที่ผูกกับ request เช่น request ID

นิยามแบบสั้นคือ Context เป็นค่าที่ใช้ส่ง deadline, cancellation signal และข้อมูลเฉพาะ request ข้ามขอบเขตของ function หรือ API เพื่อให้งานในแต่ละชั้นหยุดหรือหมดอายุไปพร้อมกับงานต้นทาง

`context.Context` เป็น interface ที่มี methods หลักดังนี้:

```go
type Context interface {
    Deadline() (deadline time.Time, ok bool)
    Done() <-chan struct{}
    Err() error
    Value(key any) any
}
```

- `Deadline()` บอกเวลาที่งานต้องสิ้นสุด หากมีการกำหนดไว้
- `Done()` คืน channel ที่ถูกปิดเมื่อ context ถูกยกเลิกหรือหมดเวลา
- `Err()` บอกสาเหตุ เช่น `context.Canceled` หรือ `context.DeadlineExceeded`
- `Value()` อ่านข้อมูลเฉพาะ request จาก key

โดยทั่วไปไม่ต้อง implement interface นี้เอง แต่สร้างและต่อยอด context ด้วย functions ใน package `context`

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

#### Deadline

`WithDeadline` กำหนดเวลาสิ้นสุดแบบเจาะจง ต่างจาก `WithTimeout` ที่กำหนดระยะเวลาสูงสุดนับจากปัจจุบัน:

```go
deadline := time.Now().Add(2 * time.Second)
ctx, cancel := context.WithDeadline(context.Background(), deadline)
defer cancel()
```

#### Cancellation

Context Cancellation คือการส่งสัญญาณให้งานที่กำลังทำอยู่ทราบว่าควรหยุด ไม่ได้บังคับฆ่า function หรือ goroutine ทันที งานหรือ library ที่รับ context ต้องร่วมมือโดยตรวจ `ctx.Done()` หรือรองรับ context ภายใน API ของตนเอง

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go doWork(ctx)
cancel()
```

- `ctx` ถูกส่งต่อไปยังงานที่ต้องการควบคุม
- `cancel()` ปิด channel ที่คืนจาก `ctx.Done()` เพื่อส่งสัญญาณยกเลิก
- หลังจากเรียก `cancel()` ค่า `ctx.Err()` จะเป็น `context.Canceled`
- ควรเรียก `cancel()` เสมอเมื่อสร้าง cancellable context เพื่อคืนทรัพยากร แม้งานจะเสร็จแล้ว

ภายในงานสามารถรอสัญญาณยกเลิกควบคู่กับผลลัพธ์ผ่าน `select`:

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

หาก function ไม่ตรวจ `ctx.Done()` และไม่ส่ง context ให้ API ที่รองรับ การเรียก `cancel()` จะไม่ทำให้ function นั้นหยุดเอง เช่น `time.Sleep` จะยังรอจนครบเวลา

สำหรับ database ไม่ต้องเขียน `select` เอง เพราะใช้ API ที่รองรับ context ได้โดยตรง:

```go
row := db.QueryRowContext(ctx, query, id)
```

เมื่อ HTTP client ยกเลิก request สัญญาณจาก `r.Context()` สามารถไหลผ่าน handler, service และ store ไปถึง database driver เพื่อให้ driver พยายามหยุด query ที่ไม่จำเป็น

Cancellation ยังส่งจาก parent ลงไปยัง child context:

```go
parent, cancelParent := context.WithCancel(context.Background())
child, cancelChild := context.WithTimeout(parent, 5*time.Second)

defer cancelParent()
defer cancelChild()
```

- ยกเลิก `parent` จะยกเลิก `child` ด้วย
- ยกเลิก `child` จะไม่ยกเลิก `parent`

ดังนั้น cancellation เป็นกลไกแบบ cooperative: ผู้ส่งแจ้งว่าควรหยุด ส่วนงานผู้รับต้องฟังสัญญาณและจบการทำงานอย่างเหมาะสม

#### Context Value

`WithValue` เหมาะสำหรับ metadata ที่เดินทางไปกับ request เช่น request ID, trace ID หรือข้อมูล authentication:

```go
type contextKey string

const requestIDKey contextKey = "request-id"

ctx := context.WithValue(parent, requestIDKey, "req-123")
requestID, ok := ctx.Value(requestIDKey).(string)
```

ควรสร้าง key type เฉพาะเพื่อลดโอกาสชนกับ key จาก package อื่น และไม่ควรใช้ Context แทน parameters ของ business logic เช่น `accountID` ควรส่งตรงผ่าน `GetAccount(ctx, accountID)`

#### ทำไมใช้ `context.Context` แทน `*context.Context`

`context.Context` เป็น interface อยู่แล้ว:

```go
type Context interface {
    Deadline() (time.Time, bool)
    Done() <-chan struct{}
    Err() error
    Value(key any) any
}
```

จึงควรรับแบบนี้:

```go
func DoSomething(ctx context.Context) error
```

ไม่ควรรับ pointer ไปยัง interface:

```go
func DoSomething(ctx *context.Context) error // ไม่แนะนำ
```

ค่า interface เก็บทั้ง concrete type และ concrete value ไว้ภายใน โดย implementation จริงอาจเป็น pointer อยู่แล้ว เช่นแนวคิดนี้:

```text
context.Context interface
    └── *cancelCtx
```

เมื่อส่ง `ctx` เข้า function Go จะ copy ค่า interface ขนาดเล็ก แต่ยังอ้างถึง context implementation เดิม สัญญาณ cancellation, deadline และ values จึงยังทำงานกับ context tree เดียวกัน ไม่ได้ copy ข้อมูลทั้งหมดแยกออกมา

`*context.Context` เป็น pointer ไปยังกล่อง interface ไม่ใช่ pointer ไปยัง implementation ภายใน ทำให้ caller ต้องส่ง `&ctx` และภายในต้อง dereference โดยแทบไม่มีประโยชน์:

```go
func DoSomething(ctx *context.Context) {
    err := (*ctx).Err()
    _ = err
}
```

หลักการเดียวกันใช้กับ interface อื่น เช่น `DBTX`:

```go
func New(db DBTX) *Queries
```

ไม่ต้องใช้ `*DBTX` เพราะ interface สามารถเก็บ concrete pointer อย่าง `*sql.DB` หรือ `*sql.Tx` อยู่ภายในได้

สรุป: รับ interface เป็น value ตามปกติ และใช้ pointer กับ concrete struct เมื่อจำเป็น ไม่ควรใช้ pointer to interface เว้นแต่มีกรณีพิเศษที่ต้องเปลี่ยนค่า interface variable ของผู้เรียกโดยตรง

#### แนวทางการใช้ Context

- รับ context เป็น parameter ตัวแรก เช่น `func DoSomething(ctx context.Context, ...)`
- รับเป็น `context.Context` ไม่ใช่ `*context.Context`
- ส่ง context เดิมต่อไปยังชั้นล่าง
- ใน HTTP handler ให้ใช้ `r.Context()` แทนการสร้าง `Background()` ใหม่
- อย่าส่ง `nil`; หากไม่มี context ให้ใช้ `context.Background()`
- โดยทั่วไปไม่ควรเก็บ context ไว้ใน struct แต่ให้รับเข้ามาในแต่ละ operation
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

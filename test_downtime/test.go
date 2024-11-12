package main

import (
    "fmt"
    "log"
    "context"
    "database/sql"
    "time"
    "strconv"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

const (
    // https://bun.uptrace.dev/postgres/#pgdriver
    dsnConn = "postgres://postgres:@localhost:54320/my_db_repl?sslmode=disable"

    //requestLoopDelay = time.Second // ~1 RPS
    requestLoopDelay = time.Millisecond * 10 // ~100 RPS
)

func main() {
    stat := struct {
        Connects, Writes, Reads, Rows int
        Failed struct {
            Connects, Writes, Reads int
        }
        Downtime struct {
            Start time.Time
            DurationMs, DurationMaxMs int
        }
    }{}

    setDowntime := func() {TEST POSTGRESQL REPLICA DOWNTIME ON MASTER DOWN
        if stat.Downtime.Start == (time.Time{}) {
            stat.Downtime.Start = time.Now()
        }
        stat.Downtime.DurationMs = int(time.Since(stat.Downtime.Start).Milliseconds())
        if stat.Downtime.DurationMaxMs < stat.Downtime.DurationMs {
            stat.Downtime.DurationMaxMs = stat.Downtime.DurationMs
        }
    }
    unsetDowntime := func() {
        stat.Downtime.Start = time.Time{}
        stat.Downtime.DurationMs = 0
    }

    var db *bun.DB

    for _ = range time.Tick(requestLoopDelay) {
        log.Printf("stat: %+v\n", stat)

        if db == nil {
            log.Printf("(re-)connecting\n")
            var err error
            if db, err = Connect(); err != nil {
                log.Printf("failed connect: %w\n", err)
                stat.Failed.Connects++
                setDowntime()
                continue
            }
            stat.Connects++
        }

        if err := Write(db); err != nil {
            log.Printf("failed write: %w\n", err)
            stat.Failed.Writes++
            setDowntime()
            db = nil
            continue
        }
        stat.Writes++

        var err error
        if stat.Rows, err = Read(db); err != nil {
            log.Printf("failed read: %w\n", err)
            stat.Failed.Reads++
            setDowntime()
            db = nil
            continue
        }
        stat.Reads++

        unsetDowntime()
    }
}

func Connect() (*bun.DB, error) {
    sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsnConn)))
    if sqldb == nil {
        return nil, fmt.Errorf("failed: nil sqldb")
    }

    db := bun.NewDB(sqldb, pgdialect.New())
    if db == nil {
        return nil, fmt.Errorf("failed: nil db")
    }

    return db, nil
}

func Write(db *bun.DB) error {
    if res, err := db.ExecContext(context.TODO(),
        "INSERT INTO test1 (data) VALUES ('row_" + strconv.Itoa(int(time.Now().UnixMilli())) + "')");
        err != nil {
        return fmt.Errorf("failed query: %v\n", err)
    } else if ra, err := res.RowsAffected(); err != nil {
        return fmt.Errorf("failed query: rows affected: %v\n", err)
    } else if ra != 1 {
        return fmt.Errorf("failed query: bad rows affected: %v\n", ra)
    }
    return nil
}

func Read(db *bun.DB) (rowsCount int, _ error) {
    sql := db.QueryRowContext(context.TODO(), "SELECT COUNT(*) FROM test1")
    if sql == nil {
        return 0, fmt.Errorf("failed query: nil sql")
    }

    count := 0
    if err := sql.Scan(&count); err != nil {
        return 0, fmt.Errorf("failed query: %w", err)
    } else if count <= 0 {
        return 0, fmt.Errorf("failed query: bad count")
    }

    return count, nil
}

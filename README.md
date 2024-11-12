
- Setup Postgresql master-slave replication (logical async)
- Setup Postgresql pgpool2
- Test Postgresql replica downtime on master down

See 01_replica.txt
    02_pgpool2.txt
    test_downtime/test.txt

Results: avg downtime duration: 100 ms
         avg failed queries: 10 (at ~100 RPS)

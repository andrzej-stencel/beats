`transaction_log` Metricset fetches information about the operation and transaction log of each MSSQL database in the monitored instance. All data is extracted from the [Database Dynamic Management Views](https://docs.microsoft.com/en-us/sql/relational-databases/system-dynamic-management-views/database-related-dynamic-management-views-transact-sql?view=sql-server-2017)

* **space_usage.since_last_backup.bytes**: The amount of space used since the last log backup in bytes
* **space_usage.total.bytes**: The size of the log in bytes
* **space_usage.used.bytes**: The occupied size of the log in bytes
* **space_usage.used.pct**: A percentage of the occupied size of the log as a percent of the total log size
* **stats.active_size.bytes**: Total active transaction log size in bytes
* **stats.backup_time**: Last transaction log backup time.
* **stats.recovery_size.bytes**: Log size in bytes since log recovery log sequence number (LSN).
* **stats.since_last_checkpoint.bytes**: Log size in bytes since last checkpoint log sequence number (LSN).
* **stats.total_size.bytes**: Total transaction log size in bytes.

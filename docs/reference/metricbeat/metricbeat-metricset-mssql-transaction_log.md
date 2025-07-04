---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-mssql-transaction_log.html
---

% This file is generated! See scripts/docs_collector.py

# MSSQL transaction_log metricset [metricbeat-metricset-mssql-transaction_log]

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

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.

## Fields [_fields]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-mssql.md) section.

Here is an example document generated by this metricset:

```json
{
    "@timestamp": "2017-10-12T08:05:34.853Z",
    "event": {
        "dataset": "mssql.transaction_log",
        "duration": 115000,
        "module": "mssql"
    },
    "metricset": {
        "name": "transaction_log",
        "period": 10000
    },
    "mssql": {
        "database": {
            "id": 1,
            "name": "master"
        },
        "transaction_log": {
            "space_usage": {
                "since_last_backup": {
                    "bytes": 937984
                },
                "total": {
                    "bytes": 2088960
                },
                "used": {
                    "bytes": 1318912,
                    "pct": 63.13725662231445
                }
            },
            "stats": {
                "active_size": {
                    "bytes": 937983.737856
                },
                "backup_time": "1900-01-01T00:00:00Z",
                "recovery_size": {
                    "bytes": 0.894531
                },
                "since_last_checkpoint": {
                    "bytes": 937983.737856
                },
                "total_size": {
                    "bytes": 2088959.475712
                }
            }
        }
    },
    "service": {
        "address": "172.23.0.2:1433",
        "type": "mssql"
    }
}
```

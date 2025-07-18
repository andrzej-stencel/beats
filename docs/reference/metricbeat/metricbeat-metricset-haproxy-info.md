---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-haproxy-info.html
---

% This file is generated! See scripts/docs_collector.py

# HAProxy info metricset [metricbeat-metricset-haproxy-info]

The HAProxy `info` metricset collects general information about HAProxy processes.

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.

## Fields [_fields]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-haproxy.md) section.

Here is an example document generated by this metricset:

```json
{
    "@timestamp": "2017-10-12T08:05:34.853Z",
    "agent": {
        "hostname": "host.example.com",
        "name": "host.example.com"
    },
    "event": {
        "dataset": "haproxy.info",
        "duration": 115000,
        "module": "haproxy"
    },
    "haproxy": {
        "info": {
            "compress": {
                "bps": {
                    "in": 0,
                    "out": 0,
                    "rate_limit": 0
                }
            },
            "connection": {
                "current": 0,
                "hard_max": 4000,
                "max": 4000,
                "rate": {
                    "limit": 0,
                    "max": 0,
                    "value": 0
                },
                "ssl": {
                    "current": 0,
                    "max": 0,
                    "total": 0
                },
                "total": 30
            },
            "idle": {
                "pct": 1
            },
            "memory": {
                "max": {
                    "bytes": 0
                }
            },
            "pipes": {
                "free": 0,
                "max": 0,
                "used": 0
            },
            "process_num": 1,
            "processes": 1,
            "requests": {
                "total": 30
            },
            "run_queue": 0,
            "session": {
                "rate": {
                    "limit": 0,
                    "max": 0,
                    "value": 0
                }
            },
            "sockets": {
                "max": 8034
            },
            "ssl": {
                "backend": {
                    "key_rate": {
                        "max": 0,
                        "value": 0
                    }
                },
                "cache_misses": 0,
                "cached_lookups": 0,
                "frontend": {
                    "key_rate": {
                        "max": 0,
                        "value": 0
                    },
                    "session_reuse": {
                        "pct": 0
                    }
                },
                "rate": {
                    "limit": 0,
                    "max": 0,
                    "value": 0
                }
            },
            "tasks": 7,
            "ulimit_n": 8034,
            "uptime": {
                "sec": 30
            },
            "zlib_mem_usage": {
                "max": 0,
                "value": 0
            }
        }
    },
    "metricset": {
        "name": "info"
    },
    "process": {
        "pid": 7
    },
    "service": {
        "address": "127.0.0.1:14567",
        "type": "haproxy"
    }
}
```

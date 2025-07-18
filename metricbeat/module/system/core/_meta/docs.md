The System `core` metricset provides usage statistics for each CPU core.

This metricset is available on:

* FreeBSD
* Linux
* macOS
* OpenBSD
* Windows


## Configuration [_configuration_4]

**`core.metrics`**
:   This option controls what metrics are reported for each CPU core. The value is a list and two metric types are supported - `percentages` and `ticks`. The default value is `core.metrics: [percentages]`.

**`use_performance_counters`**
:   This option enables the use of performance counters to collect data for the CPU/core metricset. It is only effective on Windows. You should use this option if running beats on machins with more than 64 cores. The default value is `use_performance_counters: true`

    ```yaml
    metricbeat.modules:
    - module: system
      metricsets: [core]
      core.metrics: [percentages, ticks]
      #use_performance_counters: false
    ```


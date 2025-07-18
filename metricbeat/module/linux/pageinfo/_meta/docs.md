::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


The pageinfo metricset reports on paging statistics as found in `/proc/pagetypeinfo`

Reported metrics are broken down by page type: DMA, DMA32, Normal, and Highmem. These types are further broken down by order, which represents zones of 2^ORDER*PAGE_SIZE. These metrics are divided into two reporting types: `buddyinfo`, which is summarized by page type, as in `/proc/buddyinfo`. `nodes` reports info broken down by memory migration type.

This information can be used to determine memory fragmentation. The kernel [buddy algorithim](https://www.kernel.org/doc/gorman/html/understand/understand009.html) will always search for the smallest page order to allocate, and if none is available, a larger page order will be split into two "buddies." When memory is freed, the kernel will attempt to merge the "buddies." If the only available pages are at lower orders, this indicates fragmentation, as buddy pages cannot be merged.

Note that page counts from `/proc/pagetypeinfo` will only display values up to 100,000.

This is the `broker` metricset of the ActiveMQ module.

The metricset provides metrics describing the monitored ActiveMQ broker, especially connected consumers, producers, memory usage, active connections and exchanged messages.

To collect data, the module communicates with a Jolokia HTTP/REST endpoint that exposes the JMX metrics over HTTP/REST/JSON (JMX key: `org.apache.activemq:brokerName=*,type=Broker`).

The broker metricset comes with a predefined dashboard:

![metricbeat activemq broker overview](images/metricbeat-activemq-broker-overview.png)

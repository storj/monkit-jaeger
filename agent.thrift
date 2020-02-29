include "jaeger.thrift"

namespace cpp jaegertracing.agent.thrift
namespace java io.jaegertracing.agent.thrift
namespace php Jaeger.Thrift.Agent
namespace netcore Jaeger.Thrift.Agent
namespace lua jaeger.thrift.agent

service Agent {
    oneway void emitBatch(1: jaeger.Batch batch)
}

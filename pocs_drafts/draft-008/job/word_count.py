from pyflink.datastream import StreamExecutionEnvironment
from pyflink.common.typeinfo import Types
from pyflink.datastream.functions import FlatMapFunction, RuntimeContext

class Tokenizer(FlatMapFunction):
    def flat_map(self, value, collector):
        for word in value.split():
            collector.collect((word, 1))

env = StreamExecutionEnvironment.get_execution_environment()
env.set_parallelism(1)

text = env.from_collection([
    "hello world",
    "hello flink",
    "flink stream processing with python"
], type_info=Types.STRING())

counts = text.flat_map(Tokenizer(), output_type=Types.TUPLE([Types.STRING(), Types.INT()])) \
             .key_by(lambda x: x[0]) \
             .sum(1)

counts.print()
env.execute("PyFlink WordCount")
 
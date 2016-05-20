Source({name:"kinesisstream"}).transform({filename: "transformers/json_converter.js", namespace: "kinesis.appbase_test", debug: true}).save({name:"appbaseapp", namespace:"test_123.appbase_test", debug:true});
//Source({name:"kinesisstream"}).save({name:"appbaseapp", namespace:"test_123.test", debug:true});

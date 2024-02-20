# monkit-jaeger

A plugin for http://github.com/spacemonkeygo/monkit that supports Jaeger.

## development

Thrift helpers are generated with:

```
thrift -r --gen 'go:package_prefix=storj.io/monkit-jaeger/gen-go/' agent.thrift
```

## License

Copyright (C) 2020 Storj Labs, Inc.
Copyright (C) 2016 Space Monkey, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

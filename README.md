# stuttgart-things/homerun-library

library module for homerun microservices.

## USAGE

<details><summary>USE SEND FUNC</summary>

```go
insecure := true
rendered := RenderBody(homeRunBodyData, messageBody)
fmt.Println(rendered) // DEBUG OUTPUT

answer, resp := internal.SendToHomerun(destination, token, []byte(rendered), insecure)

```

</details>

## DEV-TASKS

```bash
task: Available tasks for this project:
* branch:        Create branch from main
* check:         Run pre-commit hooks
* commit:        Commit + push code into branch
* lint:          Lint Golang
* pr:            Create pull request into main
* release:       Push new version
* tag:           Commit, push & tag the module
* test:          Test code
```

## AUTHOR

```bash
Sina Schlatter, stuttgart-things 12/2024
Patrick Hermann, stuttgart-things 10/2024
```

## License

Licensed under the Apache License, Version 2.0 (the "License").

You may obtain a copy of the License at [apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0).

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an _"AS IS"_ basis, without WARRANTIES or conditions of any kind, either express or implied.

See the License for the specific language governing permissions and limitations under the License.

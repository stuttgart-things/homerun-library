# stuttgart-things/homerun-library

library module for shared use for homerun microservice family.

## USAGE

<details><summary>USE SEND FUNC</summary>

```go
package main

import (
	"fmt"
	"time"
	homerun "github.com/stuttgart-things/homerun-library"
)

var (
	destination = "https://homerun.homerun-dev.example.com/generic"
	token       = ""
	insecure    = true
	dt          = time.Now()
)

func main() {

	// CREATE THE MESSAGE STRUCTURE
	messageBody := homerun.Message{
		Title:           "Test",
		Message:         "Test message",
		Severity:        "INFO",
		Author:          "elvis",
		Timestamp:       dt.Format("01-02-2006 15:04:05"),
		System:          "golang",
		Tags:            "golang,tests",
		AssigneeAddress: "",
		AssigneeName:    "",
		Artifacts:       "",
		Url:             "",
	}

	// RENDER THE MESSAGE BODY
	rendered := homerun.RenderBody(homerun.HomeRunBodyData, messageBody)
	fmt.Println(rendered)

	// SEND THE MESSAGE
	answer, resp := homerun.SendToHomerun(destination, token, []byte(rendered), insecure)

	// PRINT THE ANSWER
	fmt.Println("ANSWER STATUS: ", resp.Status)
	fmt.Println("ANSWER BODY: ", string(answer))
}
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

## AUTHORS

```bash
Sina Schlatter, stuttgart-things 12/2024
Patrick Hermann, stuttgart-things 10/2024
```

## License

Licensed under the Apache License, Version 2.0 (the "License").

You may obtain a copy of the License at [apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0).

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an _"AS IS"_ basis, without WARRANTIES or conditions of any kind, either express or implied.

See the License for the specific language governing permissions and limitations under the License.

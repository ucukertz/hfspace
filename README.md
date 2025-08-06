# HFSpace

**HFSpace** is a lightweight Go client for interacting with [Hugging Face Spaces](https://huggingface.co/spaces) that expose Gradio-style APIs.

It simplifies the typical two-step interaction pattern — where a POST request returns an event ID, followed by a second GET request to retrieve the actual result — into a single, streamlined call.

---

## ✨ Features

- 🌐 Connect to any Hugging Face Space endpoint  
- 🔁 Handles event ID + result fetching  
- 🔐 Supports custom headers and bearer tokens  
- ⚙️ Generic over input and output types  
- 🧼 Minimal, ergonomic API — just call `.Do()` and you're done  
- 🛡️ No dependencies outside the standard library

---

## 🚀 Example

Let's say your inference URL is `https://your-space.hf.space/gradio_api/call/fn`

```go
package main

import (
	"fmt"

	"github.com/yourusername/hfspace"
)

func main() {
	space := hfspace.NewHFSpace[any, any]("your-space").
		WithBearerToken("your-token")

	output, err := space.Do("/fn", param1, param2, param3)
	if err != nil {
		panic(err)
	}

	for _, item := range output {
		fmt.Println(item)
	}
}
```

## 🛠️ Use Cases

- Easier calls to Hugging Face Spaces from Go applications
- Integrate Spaces into backend services or pipelines
- Skip boilerplate HTTP logic and focus on Hugging Face Spaces results

---

## 📦 Installation

```bash
go get github.com/ucukertz/hfspace
```

# Linager - Go Code Lineage Analysis Tool

[![Go Report Card](https://goreportcard.com/badge/github.com/viant/linager)](https://goreportcard.com/report/github.com/viant/linager)
[![GoDoc](https://godoc.org/github.com/viant/linager?status.svg)](https://godoc.org/github.com/viant/linager)

Linager is a sophisticated static analysis tool for Go that tracks data lineage, dependencies, and usage patterns within your codebase. 
It provides detailed insights about how data flows through your application, making it easier to understand, refactor, and maintain complex code.

## Features

- **Data Flow Analysis**: Track where variables and fields are defined, read, and written
- **Dependency Tracking**: Identify data dependencies between variables and functions
- **Conditional Execution**: Capture conditions under which data is accessed or modified
- **Struct Field Analysis**: Detailed tracking of struct fields and their usage
- **Generic Types Support**: Full analysis of generic type parameters and instantiations
- **Call Graph Analysis**: Understand transitive dependencies through function calls
- **Rich Metadata**: Capture additional context like struct tags and parameter info



## Contributing

Contributions to Linager are welcome! Please feel free to submit a Pull Request.

## License


The source code is made available under the terms of the Apache License, Version 2, as stated in the file `LICENSE`.

Individual files may be made available under their own specific license,
all compatible with Apache License, Version 2. Please see individual files for details.

## Credits

Developed and maintained by [Viant](https://github.com/viant).


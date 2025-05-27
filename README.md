# DotENV

DotENV is a lightweight, zero-dependency Go utility package for loading environment variables from `.env` files. It was originally developed as part of a larger project but has proven useful enough on its own to warrant a standalone release.


## Features

- **Simple API**: Load environment variables from a single function call
- **Comprehensive syntax support**:
  - Quoted values with proper escaping: `KEY="escaped\nstring"`
  - Literal strings with single quotes: `KEY='literal value'`
  - Unquoted values: `KEY=value`
  - Comments with proper handling: `KEY=value # This is a comment`
- **Variable interpolation**: `DSN=$DB_DRIVER://$DB_USERNAME:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_DATABASE`
- **Optional overriding**: Choose whether to overwrite existing environment variables
- **Zero dependencies**: Pure Go implementation with no external packages

### Escape Sequences (in double quotes)
```
NEWLINES="Line 1\nLine 2"     # Literal \n becomes a newline character (Output: Line 1
Line 2)
TABS="Column 1\tColumn 2"     # Literal \t becomes a tab (Output: Column 1	Column 2)
QUOTES="Say \"hello\""        # Escaped quotes inside double-quoted values (Output: Say "hello")
LITERAL_DOLLAR="Cost: \$10"   # Escaped $ to avoid variable expansion (Output: Cost: $10)
```

### Variable Expansion
```
DB_DRIVER=pgsql
DB_HOST=postgres
DB_PORT=5432
DB_DATABASE=postgres
DB_USERNAME=brad
DB_PASSWORD="p4\$\$w0rd"
DB_SCHEMA=public
DB_DSN=$DB_DRIVER://$DB_USERNAME:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_DATABASE

# Output: pgsql://postgres:p4$$w0rd@postgres:5432/postgres
```

### Comments
```
# This is a full-line comment (Output: Will be ignored)
KEY=value # This is an inline comment (Output: value)
KEY="value with # inside quotes" # This comment is stripped (Output: value with # inside quotes)
KEY=when_a_string_is_not_quoted#everything_after_the_first_#_is_treated_as_a_comment (Output: when_a_string_is_not_quoted)
```


## Basic Usage

### Loading Environment Variables
```
ENV=development
```
```
package main

import (
    "fmt"
    "os"

    env "github.com/bradlilley/dotenv"
)

func main() {
    // Simulate an existing OS environment variable
    os.Setenv("ENV", "local")

    // Load and override environment variables from .env
    err := env.Load(".env", true)
    if err != nil {
        fmt.Println("Error loading .env file:", err)
    }

    env := os.Getenv("ENV")
    
    fmt.Printf("Environment: %q\n", env)
    
    // Output: Environment: "development" (Note that the environment is now development)
    
}
```

### Parsing Without Modifying the Environment
```
package main

import (
    "fmt"

    env "github.com/bradlilley/dotenv"
)

func main() {
    // Parse .env file into a map without touching the OS environment
    envMap, err := dev.Parse(".env")
    if err != nil {
        fmt.Println("Error parsing .env file:", err)
        return
    }

    // Work with the parsed variables
    for key, value := range envMap {
        fmt.Printf("%s=%s\n", key, value)
    }
}
```

## To Do
- [ ] Add topological sorting for O(n) variable expansion with complex dependency chains
- [ ] Port existing tests and benchmarks from the parent project
- [ ] Add more comprehensive documentation and examples

## License
Copyright &copy; 2025 Brad Lilley. Licensed under the [Apache License, Version 2.0](https://github.com/bradlilley/dotenv/blob/main/LICENSE).

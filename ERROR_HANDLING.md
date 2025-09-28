# Enhanced Error Handling

This document describes the enhanced error handling system in go-restkit, which provides a three-tier classification system while maintaining backward compatibility.

## Overview

The enhanced error model categorizes errors into three distinct types:

1. **Internal Library Errors** - Errors occurring during request construction within the client library
2. **Infrastructure Errors** - Errors occurring during network transmission
3. **API Errors** - Errors from server responses with structured error bodies

## Error Types

### InternalError

Represents errors in request construction that occur within the client library.

```go
type InternalError struct {
    Err error  // The underlying error
    Op  string // Operation where error occurred
}
```

**Examples:**
- JSON marshaling failures
- Invalid URL construction
- Context deadline exceeded before request transmission

**Usage:**
```go
err := client.Do(ctx, "POST", "/api", nil, payload, nil)
if rest.IsInternalError(err) {
    log.Printf("Internal error: %v", err)
}
```

### InfrastructureError

Represents network-level failures that occur during HTTP request transmission.

```go
type InfrastructureError struct {
    Err error // The underlying error
    URL string // The URL that failed to connect
}
```

**Examples:**
- Connection timeouts
- DNS resolution failures
- TLS handshake errors
- Network connectivity issues

**Usage:**
```go
err := client.Do(ctx, "GET", "https://api.example.com/data", nil, nil, nil)
if rest.IsInfrastructureError(err) {
    log.Printf("Network error contacting %s: %v", err.(*rest.InfrastructureError).URL, err)
}
```

### APIError

Represents server responses with error status codes and provides access to the response body.

```go
type APIError struct {
    StatusCode int           // HTTP status code
    URL        string        // The URL that returned the error
    Body       []byte        // Raw error response body
    Parsed     interface{}   // Optional parsed error structure
}
```

**Features:**
- Access to raw error response body via `RawBody()`
- JSON parsing of error body via `ParseError()`
- Implements `ErrorWithBody` interface

**Usage:**
```go
err := client.Do(ctx, "GET", "/api/users/123", nil, nil, nil)
if apiErr, ok := rest.AsAPIError(err); ok {
    // Access raw error body
    rawBody := apiErr.RawBody()
    
    // Parse into custom error structure
    var customErr struct {
        Code    string `json:"code"`
        Message string `json:"message"`
    }
    if parseErr := apiErr.ParseError(&customErr); parseErr == nil {
        log.Printf("API error %s: %s", customErr.Code, customErr.Message)
    }
}
```

## Error Detection Functions

### Type-Specific Detection

```go
// Check for specific error types
if rest.IsInternalError(err) {
    // Handle internal library errors
}

if rest.IsInfrastructureError(err) {
    // Handle network infrastructure errors
}

if rest.IsAPIError(err) {
    // Handle API response errors
}
```

### Error Extraction

```go
// Extract APIError from error chain
if apiErr, ok := rest.AsAPIError(err); ok {
    // Work with the specific APIError
    fmt.Printf("Status: %d, URL: %s", apiErr.StatusCode, apiErr.URL)
}
```

## Best Practices

### Error Handling

1. **Check error types specifically** rather than relying on error strings
2. **Use structured error parsing** for API errors when possible
3. **Provide meaningful context** in error logs
4. **Handle different error types appropriately** in your application logic

### Logging

```go
err := client.Do(ctx, "GET", "/api/data", nil, nil, &response)
if err != nil {
    if apiErr, ok := rest.AsAPIError(err); ok {
        log.Printf("API request failed: %s %s - %s", 
            apiErr.StatusCode, apiErr.URL, string(apiErr.RawBody()))
    } else {
        log.Printf("Request failed: %v", err)
    }
}
```

### User-Friendly Messages

```go
err := client.Do(ctx, "GET", "/api/data", nil, nil, &response)
if err != nil {
    var userMsg string
    
    if apiErr, ok := rest.AsAPIError(err); ok {
        var apiResp struct {
            Message string `json:"message"`
        }
        if apiErr.ParseError(&apiResp) == nil {
            userMsg = apiResp.Message
        } else {
            userMsg = fmt.Sprintf("Request failed with status %d", apiErr.StatusCode)
        }
    } else if rest.IsInfrastructureError(err) {
        userMsg = "Network connection error. Please check your internet connection."
    } else {
        userMsg = "An unexpected error occurred."
    }
    
    fmt.Println(userMsg)
}
```

## Testing

When testing error handling, consider all three error types:

```go
func TestErrorHandling(t *testing.T) {
    // Test internal error (invalid payload)
    err := client.Do(ctx, "POST", "/api", nil, math.NaN(), nil)
    if !rest.IsInternalError(err) {
        t.Error("Expected internal error for invalid payload")
    }

    // Test infrastructure error (unreachable host)
    client := rest.NewClient(rest.Config{BaseURL: "http://localhost:1"})
    err = client.Do(ctx, "GET", "/", nil, nil, nil)
    if !rest.IsInfrastructureError(err) {
        t.Error("Expected infrastructure error for unreachable host")
    }

    // Test API error (server response)
    server := setupTestServer(t)
    client = rest.NewClient(rest.Config{BaseURL: server.URL})
    err = client.Do(ctx, "GET", "/400", nil, nil, nil)
    
    if apiErr, ok := rest.AsAPIError(err); ok {
        if apiErr.StatusCode != 400 {
            t.Errorf("Expected status 400, got %d", apiErr.StatusCode)
        }
    } else {
        t.Error("Expected API error")
    }
}
```

## Summary

The enhanced error model provides:

- **Clear error classification** with three distinct error types
- **Rich error information** including response bodies and status codes
- **Better testing capabilities** with specific error type detection
- **Improved error handling** for more robust applications

This enhancement makes it easier to diagnose and handle errors in your applications while maintaining compatibility with existing code.
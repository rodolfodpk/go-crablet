# Code Coverage Analysis

Current test coverage status and improvement guidelines for the go-crablet library.

## 📊 **Current Status**

| Package | Coverage | Status |
|---------|----------|---------|
| **pkg/dcb** (Core Library) | **85.7%** | ✅ Good |

### **Coverage by Function Type**

- **Event Store Operations**: 78-92% (Read, Append, NewEventStore)
- **Streaming Operations**: 70-100% (ReadStream, Next, Event, Close)
- **Projection Operations**: 84-100% (Project, combineProjectorQueries)
- **Validation Functions**: 100% (validateQueryTags, validateEvent, etc.)
- **Helper Functions**: 75-100% (NewTags, NewQuery, toJSON)

## 🎯 **Goals**

- **Short-term**: 90%+ core library coverage
- **Long-term**: 95%+ overall coverage
- **Priority**: Improve streaming operations (70% → 90%)

## 🧪 **Testing Commands**

```bash
# Run tests with coverage
go test -coverprofile=coverage.out ./pkg/dcb

# View detailed coverage
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# Run with race detection
go test -race ./pkg/dcb
```

## 📈 **Improvement Guidelines**

### **Adding Tests**
1. Follow existing test patterns
2. Test success and failure cases
3. Include edge cases and boundary conditions
4. Test concurrent operations where applicable

### **Test Quality**
- Descriptive test names
- Clear arrange/act/assert structure
- Comprehensive assertions
- Proper resource cleanup

## 🔍 **Coverage Exclusions**

- `internal/benchmarks/` - Performance benchmarks
- `internal/examples/` - Example applications
- `docs/` - Documentation files
- `cmd/` - Command-line tools

## 🚨 **Critical Paths (Target: 100%)**

- Event appending with optimistic locking
- Event reading and querying
- Decision model projection
- Input validation and error handling
- Streaming operations and resource cleanup

## 📊 **Monitoring**

```yaml
# CI coverage check
- name: Check Coverage Threshold
  run: |
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    if (( $(echo "$COVERAGE < 85" | bc -l) )); then
      echo "Coverage below threshold: $COVERAGE%"
      exit 1
    fi
```

---

**Focus**: Improve streaming operation coverage and maintain high coverage for new features. 
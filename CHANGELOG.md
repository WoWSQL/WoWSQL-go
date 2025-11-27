# Changelog

All notable changes to the WOWSQL Go SDK will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2025-11-22

### Added - Schema Management 🔧

- **New `SchemaClient` struct** for programmatic database schema management
- Full schema CRUD operations with service role key authentication
- Schema modification capabilities for production databases

#### Schema Features
- **Create Tables**: Define tables with columns, primary keys, and indexes
- **Alter Tables**: Add, modify, drop, or rename columns
- **Drop Tables**: Remove tables with optional CASCADE support
- **Execute SQL**: Run raw SQL for custom schema operations

#### Schema API Methods
- `CreateTable()` - Create new tables with full column definitions
- `AlterTable()` - Modify existing table structure
- `DropTable()` - Drop tables safely
- `ExecuteSQL()` - Execute custom schema SQL statements

#### Security & Validation
- **Service Role Key Required**: Schema operations strictly require service role keys
- **403 Error Handling**: Clear error messages when using anonymous keys
- **Permission Validation**: Automatic validation of API key permissions

#### Go Structs
- `ColumnDefinition` - Column specification with constraints
- `CreateTableRequest` - Table creation configuration
- `AlterTableRequest` - Table alteration specification
- `SchemaResponse` - Schema operation response
- Complete type definitions for all schema operations

#### Examples & Documentation
- Backend migration script examples
- Schema management best practices
- Security guidelines for service key usage
- Comprehensive README section with code examples

### Updated
- README with comprehensive schema management documentation
- Version bumped to 1.2.0

## [1.1.0] - 2025-11-11

### Added - API Keys Documentation

- Comprehensive API keys documentation section in README
- Clear separation between Database Operations keys and Authentication Operations keys
- Documentation for Service Role Key, Public API Key, and Anonymous Key
- Usage examples for each key type
- Security best practices guide
- Troubleshooting section for common key-related errors

### Updated

- Enhanced `WOWSQLClient` class documentation to clarify it's for DATABASE OPERATIONS
- Enhanced `ProjectAuthClient` class documentation to clarify it's for AUTHENTICATION OPERATIONS
- Version bumped to 1.1.0

### Documentation

- README now includes comprehensive API keys section with:
  - Key types overview table
  - Where to find keys in dashboard
  - Database operations examples
  - Authentication operations examples
  - Environment variables best practices
  - Security best practices
  - Troubleshooting guide

## [1.0.0] - 2025-10-17

### Added - Initial Release 🎉

#### Database Client
- **Full CRUD operations** - Create, Read, Update, Delete
- **Fluent query builder** - Chainable API for intuitive queries
- **Advanced filtering** - eq, neq, gt, gte, lt, lte, like, isNull
- **Pagination** - limit and offset support
- **Sorting** - orderBy with asc/desc directions
- **Raw SQL queries** - Execute custom SQL
- **Table introspection** - List tables and get schema information
- **Health check** - Check API status
- **Type-safe** - Go's type system with structs
- **Idiomatic Go** - Following Go conventions and best practices

#### Storage Client
- **S3-compatible storage** - Full file management
- **File upload** - Upload files with automatic quota validation
- **File download** - Get presigned URLs for downloads
- **File listing** - List files with optional prefix filtering
- **File deletion** - Delete single or multiple files
- **Quota management** - Check storage usage and limits
- **File info** - Get detailed file metadata
- **Multi-region** - Support for different S3 regions
- **Client-side validation** - Prevent uploads exceeding quota

#### Error Handling
- `WOWSQLError` - Base error for database errors
- `AuthenticationError` - Authentication errors (401/403)
- `NotFoundError` - Not found errors (404)
- `RateLimitError` - Rate limit errors (429)
- `NetworkError` - Network connectivity errors
- `StorageError` - Base error for storage errors
- `StorageLimitExceededError` - Storage quota exceeded (413)
- Standard Go error handling with `errors.As()`

#### Models
- `QueryResponse` - Query response with data and count
- `CreateResponse` - Response from insert operations
- `UpdateResponse` - Response from update operations
- `DeleteResponse` - Response from delete operations
- `TableSchema` - Table schema information
- `ColumnInfo` - Column metadata
- `StorageQuota` - Storage quota information
- `StorageFile` - File metadata
- `FileUploadResult` - Upload operation result

#### Features
- 🚀 Zero configuration required
- 🔒 Secure API key authentication
- ⚡ Fast and efficient with net/http
- 🛡️ Comprehensive error handling
- 📝 Full GoDoc documentation
- 🎯 Idiomatic Go code
- ✅ Production ready
- 🔄 Context support (planned for v1.1.0)

### Documentation
- Complete README with usage examples
- API reference documentation (GoDoc)
- Error handling guide
- Publishing guide for pkg.go.dev

---

## Future Roadmap

### Planned Features (v1.1.0)

- [ ] Context support for cancellation and timeouts
- [ ] Batch operations for database
- [ ] Transaction support
- [ ] Query caching
- [ ] Retry logic with exponential backoff
- [ ] Streaming uploads for large files
- [ ] File versioning support
- [ ] Aggregation functions (COUNT, SUM, AVG)
- [ ] Connection pooling optimization

### Under Consideration

- [ ] Real-time subscriptions (WebSocket)
- [ ] Code generation for models from schema
- [ ] Migration tools
- [ ] GraphQL-like nested queries
- [ ] Offline-first support with sync
- [ ] gRPC support

---

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](../../CONTRIBUTING.md) for details.

## Support

- 📧 Email: support@wowsql.com
- 💬 Discord: [Join our community](https://discord.gg/WOWSQL)
- 📚 Documentation: [https://wowsql.com/docs](https://wowsql.com/docs)
- 🐛 Issues: [GitHub Issues](https://github.com/wowsql/wowsql/issues)

---

For more information, visit [https://wowsql.com](https://wowsql.com)


# Changelog

All notable changes to the Multi-Tenant Hybrid Storage System will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2025-07-01

### 🎉 Major Release - Microservices Architecture Migration

This is a complete architectural transformation from a monolithic system to a microservices-based architecture.

### ✨ Added

#### New Microservices
- **Auth Gateway** (8080) - JWT-based authentication and API gateway
- **Operation Node** (8081) - Tenant coordination and auto-scaling
- **CBO Engine** (8082) - Cost-based query optimization
- **Metadata Catalog** (8083) - Metadata management and compaction
- **Monitoring** (8084) - Observability and health monitoring
- **Query Interpreter** (8085) - SQL/DSL parsing and query plan generation

#### Query Processing Capabilities
- **SQL Parser** using SQLGlot with support for:
  - Complex SELECT statements with JOINs
  - Aggregations (SUM, COUNT, AVG, MIN, MAX, GROUP BY)
  - Subqueries and Common Table Expressions (CTEs)
  - Window functions and analytical operations
- **DSL Support** for custom domain-specific queries
- **Query Transformation** pipeline (Parse → Validate → Transform → Optimize)
- **Query Plan Generation** for execution optimization

#### Authentication & Security
- **JWT-based Authentication** with configurable expiration
- **Role-based Access Control (RBAC)** for fine-grained permissions
- **Multi-tenant Security** with tenant isolation
- **API Key Management** for programmatic access

#### Development & Operations
- **Docker Compose** orchestration for all services
- **Service-specific Dockerfiles** for containerized deployment
- **Microservices Demo Scripts**:
  - `microservices_demo.py` - Integration testing
  - `auth_demo.py` - Authentication flow testing
  - `scaling_demo.py` - Auto-scaling demonstration
- **Convenience Scripts**:
  - `start_services.ps1` / `start_services.sh` - Start all services
  - `stop_services.sh` - Stop all services
  - `cleanup.ps1` / `cleanup.sh` - Development environment cleanup
- **Documentation**:
  - `QUICKSTART.md` - Quick start guide for new users
  - `MIGRATION.md` - Migration guide from v1.x
  - Service-specific README files for each microservice

#### Configuration Management
- **Service-specific Environment Variables** for each microservice
- **Docker Compose Override** support for local customization
- **Configuration Files** (JSON) for complex service settings
- **Environment-specific Configurations** (dev, staging, prod)

### 🔄 Changed

#### Architecture
- **Monolithic → Microservices**: Complete system redesign
- **Single Entry Point → Service-Oriented**: Each service has its own API
- **Centralized → Distributed**: Services communicate via REST/gRPC
- **File-based → Service-based**: Configuration distributed across services

#### API Structure
- **Single REST API → Multiple Service APIs**: Each service exposes its own endpoints
- **Port Consolidation → Port Distribution**: Services run on dedicated ports (8000-8085)
- **Direct Access → Gateway Pattern**: Auth Gateway handles authentication and routing

#### Data Processing
- **Embedded Query Processing → Dedicated Service**: Query Interpreter service handles SQL/DSL
- **Basic Optimization → Advanced CBO**: Machine learning-enhanced cost-based optimization
- **Manual Scaling → Auto-scaling**: Operation Node manages dynamic resource allocation

### 📦 Repository Structure
- **Service Isolation**: Each microservice in its own directory with dependencies
- **Legacy Preservation**: Old demos moved to `legacy_demos/` for reference
- **Clean Separation**: Shared components in `shared-protos/` directory

### 🗑️ Removed

#### Obsolete Files
- `main.py` - Replaced by service-specific main.py files
- `run.py` - Replaced by Docker Compose and service scripts
- `demo.py` - Replaced by microservices-specific demo scripts
- `advanced_demo.py` - Archived in `legacy_demos/`
- `config.env.example` - Replaced by service-specific configurations
- `requirements.txt` - Replaced by service-specific requirements
- `setup.bat` / `setup.sh` - Replaced by Docker Compose setup
- `instructions.txt` - Content integrated into documentation

#### Legacy Components
- Monolithic tenant node implementation
- Single-service configuration approach
- Direct file-based configuration management

### 🛠️ Technical Improvements

#### Performance
- **Parallel Service Execution**: Services can scale independently
- **Optimized Query Processing**: Dedicated query interpretation and optimization
- **Efficient Resource Usage**: Auto-scaling based on actual load

#### Maintainability
- **Service Isolation**: Changes to one service don't affect others
- **Independent Deployments**: Services can be deployed and updated independently
- **Clear Boundaries**: Well-defined service responsibilities and APIs

#### Observability
- **Service-specific Monitoring**: Each service can be monitored independently
- **Distributed Tracing**: End-to-end request tracking across services
- **Comprehensive Health Checks**: Service and system-wide health monitoring

### 🚀 Deployment

#### Container Support
- **Docker Compose**: Single-command deployment for development
- **Kubernetes Ready**: Service definitions ready for K8s deployment
- **Load Balancer Support**: NGINX configuration examples provided

#### Environment Support
- **Development**: Docker Compose with local volumes
- **Staging**: Multi-instance deployment with external dependencies
- **Production**: Scalable deployment with monitoring and observability

### 📖 Documentation

#### User Guides
- **README.md**: Completely rewritten for microservices architecture
- **QUICKSTART.md**: Step-by-step setup guide
- **MIGRATION.md**: Guide for migrating from v1.x

#### API Documentation
- **Service-specific APIs**: Detailed endpoint documentation for each service
- **Authentication Flow**: Complete auth workflow examples
- **Query Processing**: SQL and DSL query examples with transformations

#### Development Guides
- **Service Development**: How to develop and test individual services
- **Integration Testing**: How to test service interactions
- **Deployment Guide**: Production deployment best practices

### 🔮 Future Readiness

#### Repository Separation
- **Independent Repositories**: Each service ready for its own Git repository
- **CI/CD Pipelines**: Service-specific build and deployment pipelines
- **Version Management**: Independent versioning for each service

#### Extensibility
- **Plugin Architecture**: Services designed for easy extension
- **Protocol Buffer Support**: gRPC-ready for high-performance communication
- **Message Queue Integration**: Ready for async processing patterns

### 🏃‍♂️ Migration Path

For users migrating from v1.x:
1. **Backup existing data**: Preserve current data and configurations
2. **Run new services**: Start with Docker Compose for testing
3. **Update integrations**: Modify client applications for new API structure
4. **Gradual migration**: Move data sources one by one
5. **Full transition**: Retire old monolithic deployment

See `MIGRATION.md` for detailed migration instructions.

---

## [1.x] - Legacy Versions

Previous versions of the monolithic architecture are archived in `legacy_demos/` for reference.

### Key Features (v1.x)
- Monolithic tenant node architecture
- Single REST API endpoint
- File-based configuration
- Basic query processing
- Manual scaling

---

## Support

For questions about this release or migration assistance:
- Check the `QUICKSTART.md` for setup instructions
- Review `MIGRATION.md` for migration guidance
- Run the demo scripts to understand the new architecture
- Refer to service-specific README files for detailed documentation

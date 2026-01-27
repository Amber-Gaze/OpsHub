# OpsHub
A practice project for learning Go.The project is divided into three submodules.  
1. Gateway: focuses on external gateway services, implementing JWT verification and rate limiting.   
2. Auth: handles authentication and authorization management.   
3. Config Center: manages production configurations.  

The three components collectively accomplish a function that enables real-time updates and deployment of online configurations.  

 
这是一个我自己用于学习Go语言的实践项目。该项目分为三个子模块：  
1. 网关：专注于外部网关服务，实现JWT验证和速率限制。  
2. 认证：处理身份验证与授权管理。  
3. 配置中心：管理生产环境配置。  

这三个组件共同实现实时更新和部署在线配置运控台。 


## I. Gateway
### Traffic Entry
1. HTTP / HTTPS
2. API Gateway
### Security Protection
1. Rate limiting
2. Allow/Deny lists
3. Basic validation
### Authorization Execution
1. Ask Auth: is it allowed?
Decide to pass or block according to Auth's response
### Context Injection
1. Write "who is acting" into request headers
2. Downstream services use it for audit
### Request Forwarding
1. Forward to Config Center
2. Forward to other business services

## II. Auth
### Identity Management
1. User registration / login / disable
2. Credentials (password / token / certificate)
### Authorization Model
1. User → Role → Permission
2. Permission definition: action + resource
### Authorization Decision
Answer one question: "Can this person perform this action on this resource?"
### Issue Authorization Result
1. Return allow / deny
2. Return a tamper-proof decision credential

## Config Center
### Configuration Read/Write
1. Key / Value
2. Groups / Projects
### Version Management
1. History versions
2. Rollback
### Audit
1. Who
2. When
3. What changed
### Anti-Forgery
1. Only accept Gateway requests
2. Validate authorization credentials
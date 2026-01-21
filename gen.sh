#!/usr/bin/env bash
set -e

MODULE="github.com/Amber-Gaze/OpsHub"

echo "==> init go module"
go mod init ${MODULE}

echo "==> create directories"
dirs=(
  cmd/api
  cmd/grpc
  internal/app/api/middleware
  internal/app/api/handler
  internal/app/api
  internal/app/grpc
  internal/domain
  internal/service
  internal/repository/mysql
  internal/repository/redis
  internal/repository/etcd
  internal/infrastructure
  internal/pkg/response
  internal/pkg/errors
  internal/pkg/utils
  internal/pkg/constants
  proto
  configs
  scripts/sql
)

for d in "${dirs[@]}"; do
  mkdir -p "$d"
done

echo "==> create files"

# ---------- cmd/api/main.go ----------
cat > cmd/api/main.go <<'EOF'
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/Amber-Gaze/OpsHub/internal/app/api"
)

func main() {
    r := gin.New()
    r.Use(gin.Recovery(), gin.Logger())

    api.RegisterRoutes(r)

    r.Run(":8080")
}
EOF

# ---------- api/router.go ----------
cat > internal/app/api/router.go <<'EOF'
package api

import (
    "github.com/gin-gonic/gin"
    "github.com/Amber-Gaze/OpsHub/internal/app/api/middleware"
    "github.com/Amber-Gaze/OpsHub/internal/app/api/handler"
)

func RegisterRoutes(r *gin.Engine) {
    r.Use(middleware.RateLimit())

    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })

    api := r.Group("/api")
    api.Use(middleware.Auth())

    api.GET("/configs", handler.ListConfigs)
}
EOF

# ---------- middleware ----------
cat > internal/app/api/middleware/auth.go <<'EOF'
package middleware

import "github.com/gin-gonic/gin"

func Auth() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Set("user", "admin")
        c.Next()
    }
}
EOF

cat > internal/app/api/middleware/ratelimit.go <<'EOF'
package middleware

import (
    "time"
    "github.com/gin-gonic/gin"
    "golang.org/x/time/rate"
)

var limiter = rate.NewLimiter(rate.Every(time.Second), 100)

func RateLimit() gin.HandlerFunc {
    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.AbortWithStatusJSON(429, gin.H{"msg": "too many requests"})
            return
        }
        c.Next()
    }
}
EOF

cat > internal/app/api/middleware/casbin.go <<'EOF'
package middleware

import "github.com/gin-gonic/gin"

func Casbin() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
    }
}
EOF

cat > internal/app/api/middleware/audit.go <<'EOF'
package middleware

import "github.com/gin-gonic/gin"

func Audit() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
    }
}
EOF

# ---------- handler ----------
cat > internal/app/api/handler/config.go <<'EOF'
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/Amber-Gaze/OpsHub/internal/service"
)

var configSvc = service.NewConfigService()

func ListConfigs(c *gin.Context) {
    data, _ := configSvc.List()
    c.JSON(http.StatusOK, data)
}
EOF

cat > internal/app/api/handler/user.go <<'EOF'
package handler

import "github.com/gin-gonic/gin"

func ListUsers(c *gin.Context) {
    c.JSON(200, gin.H{"users": []string{}})
}
EOF

cat > internal/app/api/handler/project.go <<'EOF'
package handler

import "github.com/gin-gonic/gin"

func ListProjects(c *gin.Context) {
    c.JSON(200, gin.H{"projects": []string{}})
}
EOF

cat > internal/app/api/handler/audit.go <<'EOF'
package handler

import "github.com/gin-gonic/gin"

func ListAudits(c *gin.Context) {
    c.JSON(200, gin.H{"audits": []string{}})
}
EOF

cat > internal/app/api/handler/metrics.go <<'EOF'
package handler

import "github.com/gin-gonic/gin"

func Metrics(c *gin.Context) {
    c.JSON(200, gin.H{"metrics": "ok"})
}
EOF

# ---------- domain ----------
cat > internal/domain/config.go <<'EOF'
package domain

type Config struct {
    Project string
    Group   string
    Key     string
    Value   string
    Version int64
}
EOF

cat > internal/domain/config_history.go <<'EOF'
package domain

import "time"

type ConfigHistory struct {
    Project   string
    Group     string
    Key       string
    Value     string
    Revision  int64
    CreatedAt time.Time
}
EOF

cat > internal/domain/user.go <<'EOF'
package domain

type User struct {
    ID   int64
    Name string
}
EOF

cat > internal/domain/project.go <<'EOF'
package domain

type Project struct {
    ID   int64
    Name string
}
EOF

cat > internal/domain/audit.go <<'EOF'
package domain

import "time"

type Audit struct {
    User      string
    Action    string
    CreatedAt time.Time
}
EOF

# ---------- service ----------
cat > internal/service/config_service.go <<'EOF'
package service

import "github.com/Amber-Gaze/OpsHub/internal/domain"

type ConfigService struct{}

func NewConfigService() *ConfigService {
    return &ConfigService{}
}

func (s *ConfigService) List() ([]domain.Config, error) {
    return []domain.Config{}, nil
}
EOF

cat > internal/service/user_service.go <<'EOF'
package service

type UserService struct{}
EOF

cat > internal/service/project_service.go <<'EOF'
package service

type ProjectService struct{}
EOF

cat > internal/service/audit_service.go <<'EOF'
package service

type AuditService struct{}
EOF

# ---------- infrastructure ----------
cat > internal/infrastructure/db.go <<'EOF'
package infrastructure

func InitDB() {}
EOF

cat > internal/infrastructure/redis.go <<'EOF'
package infrastructure

func InitRedis() {}
EOF

cat > internal/infrastructure/etcd.go <<'EOF'
package infrastructure

func InitEtcd() {}
EOF

cat > internal/infrastructure/zap.go <<'EOF'
package infrastructure

func InitZap() {}
EOF

# ---------- proto ----------
cat > proto/service.proto <<'EOF'
syntax = "proto3";
package proto;

service ConfigService {}
EOF

echo "==> done"


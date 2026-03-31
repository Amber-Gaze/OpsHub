package casbinx

// ModelText Casbin 模型：RBAC + globMatch（obj 形如 config/模块/键 或 config/pay/*）。
const ModelText = `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = (g(r.sub, p.sub) && globMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")) || (r.sub == p.sub && globMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*"))
`

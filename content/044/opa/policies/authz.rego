package authz

import rego.v1

default allow := false

# Admins podem tudo
allow if {
    input.subject.role == "admin"
}

# Usuário só pode ler seus próprios pedidos
allow if {
    input.action == "read"
    input.resource.type == "order"
    input.resource.owner == input.subject.id
}

# Usuário pode criar pedidos
allow if {
    input.action == "create"
    input.resource.type == "order"
    input.subject.role in {"admin", "user"}
}

# Motivo estruturado (útil p/ audit log)
reason := "allowed: admin" if input.subject.role == "admin"
reason := "allowed: owner reading own order" if {
    input.action == "read"
    input.resource.owner == input.subject.id
}
reason := "denied: not owner and not admin" if not allow

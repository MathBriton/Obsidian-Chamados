# Obsidian Chamados

SaaS Multi-Tenant de Gestão de Chamados (Help Desk / Service Desk), inspirado em Jira Service Management, Zendesk e Freshdesk. Projeto de portfólio com foco em aprendizado técnico, **Spec Driven Development** e **TDD**.

## Stack

| Camada | Tecnologias |
|---|---|
| Backend | Go, Gin, SQLite, SQLC, JWT |
| Frontend | React, TypeScript, React Router, Tailwind |
| Testes | `testing`, `httptest`, Vitest, React Testing Library |
| Infra | Docker (multi-stage, distroless), GitHub Actions (CI/CD) |

## Arquitetura

```
cmd/server/          # entrypoint
internal/
  auth/              # JWT + bcrypt
  config/            # variáveis de ambiente
  database/          # conexão SQLite + migrations
  db/                # código gerado pelo SQLC (não editar à mão)
  handlers/          # HTTP (Gin)
  middleware/        # autenticação, tenant
  models/            # erros e tipos de domínio
  repositories/      # acesso a dados (envolve o SQLC)
  services/          # regras de negócio
migrations/          # SQL versionado (golang-migrate)
sql/queries/         # queries do SQLC
```

## Decisões Arquiteturais (ADRs)

- **ADR-001** — Isolamento multi-tenant via coluna `tenant_id` em todas as tabelas de negócio (banco único). Reforçado na camada de repositório: toda query filtra por `tenant_id`.
- **ADR-002** — IDs inteiros sequenciais (`AUTOINCREMENT`). O isolamento depende 100% do filtro `WHERE tenant_id = ?`, coberto por testes anti cross-tenant.
- **ADR-003** — Acesso cross-tenant retorna **404**, nunca 403 (não revela existência do recurso).
- **ADR-004** — Driver SQLite **pure-Go** (`modernc.org/sqlite`), sem CGO. Simplifica build, CI e imagem Docker.

## Guia de Instalação

### Pré-requisitos

| Ferramenta | Versão | Observação |
|---|---|---|
| Go | 1.26+ | Build do backend (sem CGO — driver SQLite pure-Go) |
| Git | qualquer | Clonar o repositório |
| Docker + Compose | opcional | Apenas para rodar em container (produção) |

> Não é necessário compilador C: o driver `modernc.org/sqlite` dispensa CGO (ADR-004).

### 1. Clonar o repositório

```bash
git clone https://github.com/MathBriton/Obsidian-Chamados.git
cd Obsidian-Chamados
```

### 2. Configurar variáveis de ambiente

Copie o template e ajuste os valores. **Nunca** versione o `.env` (já está no `.gitignore`).

```bash
cp .env.example .env
```

| Variável | Default (dev) | Descrição |
|---|---|---|
| `PORT` | `8080` | Porta HTTP do servidor |
| `DATABASE_URL` | `file:chamados.db?_pragma=foreign_keys(1)` | DSN do SQLite |
| `JWT_SECRET` | `dev-secret-change-me` | **Troque em produção** por um valor forte e aleatório |

> Gerar um segredo forte: `openssl rand -hex 32`

### 3. Baixar dependências

```bash
go mod download
```

### 4. Rodar o servidor

```bash
go run ./cmd/server
```

As migrations são aplicadas automaticamente no boot. O servidor sobe em `http://localhost:8080`. Verifique:

```bash
curl http://localhost:8080/healthz
# {"status":"ok"}
```

## API — Autenticação

| Método | Rota | Auth | Descrição |
|---|---|---|---|
| `POST` | `/auth/register` | — | Cria tenant + usuário admin, retorna par de tokens |
| `POST` | `/auth/login` | — | Autentica dentro do tenant (por `slug`) |
| `POST` | `/auth/refresh` | — | Rotaciona o refresh token e emite novo par |
| `POST` | `/auth/logout` | — | Revoga o refresh token (idempotente) |
| `GET`  | `/me` | Bearer | Retorna o usuário autenticado |
| `GET`  | `/healthz` | — | Liveness probe |

### Categorias e Tickets

| Método | Rota | Auth | Descrição |
|---|---|---|---|
| `GET`  | `/categories` | Bearer | Lista as categorias do tenant |
| `POST` | `/categories` | Bearer (admin) | Cria uma categoria |
| `POST` | `/tickets` | Bearer | Abre um ticket (qualquer papel) |
| `GET`  | `/tickets` | Bearer | Lista tickets (`?limit=&offset=`) |
| `GET`  | `/tickets/:id` | Bearer | Detalha um ticket |
| `PATCH`| `/tickets/:id` | Bearer | Atualização parcial |
| `POST` | `/tickets/:id/comments` | Bearer | Comenta no ticket (`is_internal` só staff) |
| `GET`  | `/tickets/:id/comments` | Bearer | Lista comentários (customer não vê notas internas) |

**Autorização (RBAC):** `customer` cria e vê/edita apenas os próprios tickets (e só `title`/`description`); `agent`/`admin` veem e editam todos do tenant, incluindo `status`, `priority`, `category_id` e `assigned_to`. Ticket de outro tenant ou de outro customer responde **404** (não revela existência — ADR-003). Transições para `resolved`/`closed` carimbam `resolved_at`/`closed_at`.

Exemplo de fluxo:

```bash
# Registrar
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"tenant_name":"Acme","slug":"acme","name":"Admin","email":"admin@acme.com","password":"senha-super-secreta"}'

# Usar o access_token retornado
curl http://localhost:8080/me -H "Authorization: Bearer <ACCESS_TOKEN>"
```

Erros seguem o formato `{"error":{"code":"...","message":"..."}}` com códigos estáveis (RNF13), ex.: `invalid_credentials`, `slug_taken`, `invalid_token`.

## Documentação interativa (Swagger)

Com o servidor rodando, a Swagger UI fica disponível em:

```
http://localhost:8080/swagger/index.html
```

É possível testar todos os endpoints pelo navegador. Para rotas autenticadas, clique em **Authorize** e informe `Bearer <access_token>` (obtido em `/auth/register` ou `/auth/login`).

A documentação é gerada por anotações [swaggo](https://github.com/swaggo/swag) nos handlers. Para regenerar após mudanças na API:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/server/main.go -o docs --parseInternal --parseDependency
```

O pacote `docs/` é versionado, então o servidor e a imagem Docker não dependem do `swag` em runtime.

## Testes

```bash
go test ./...              # suíte completa
go test -race ./...        # com detector de race (exige CGO; usado no CI)
go test -cover ./...       # com cobertura
```

Testes de integração usam SQLite em memória (RNF10) — não tocam o banco real.

## Docker (produção)

Imagem multi-stage com runtime **distroless non-root**:

```bash
# JWT_SECRET é lido do ambiente; defina antes em produção
export JWT_SECRET=$(openssl rand -hex 32)
docker compose up --build
```

O SQLite é persistido no volume `chamados-data`.

## Metodologia

Toda funcionalidade segue a ordem: **Requisitos → Casos de Uso → Modelo de Dados → API Contract → Testes → Implementação**.

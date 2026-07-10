# 🚀 Mini E-Commerce - Ecossistema de Microsserviços em Go

Este repositório contém a implementação de uma arquitetura orientada a microsserviços para um ecossistema de e-commerce resiliente e de alta performance, desenvolvido inteiramente em **Golang**. O projeto foca no isolamento completo de domínios, auditoria transparente de dados, consistência eventual orientada a eventos e no uso intensivo da biblioteca padrão para máxima eficiência.

---

## 📐 Arquitetura de Infraestrutura & Fluxo

O ecossistema é composto por quatro microsserviços de negócio apoiados por um **API Gateway** central e um **Message Broker**.

```text
               ┌───────────────────────────────────┐
               │        API Gateway (Porta 8080)   │
               │ (Chi Router + gobreaker + Proxy)  │
               └──────┬───┬─────────────┬───┬──────┘
                      │   │             │   │
/api/auth/**    /api/product/**  /api/orders/**  /api/stock/**
                      │   │             │   │
                      ▼   ▼             ▼   ▼
┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐
│     ms_auth      │ │    ms_product    │ │     ms_order     │ │     ms_stock     │
│   (Porta 4001)   │ │   (Porta 4002)   │ │   (Porta 4004)   │ │   (Porta 4003)   │
└─────────┬────────┘ └─────────┬────────┘ └─────────┬────────┘ └─────────┬────────┘
          │                    │                    │                    │
          ▼                    ▼                    │                    ▼
     [ PostgreSQL ]      [ PostgreSQL ]             │             [ PostgreSQL ]
                                               Kafka
                                        (Async Event: OrderCreated)
```

### 🎛️ Componentes Centrais de Infraestrutura
1. **API Gateway (`ms_gateway` - Porta 8080):** Único ponto de entrada para o tráfego externo. Implementado usando `httputil.ReverseProxy` nativo e roteador `go-chi`. Isola falhas locais usando **Circuit Breaker (`gobreaker`)** e reescreve as rotas de forma transparente (`/api/...` para `/v1/...`).
2. **Gerenciamento de Configuração:** Seguindo os princípios da aplicação 12-Factor, não utilizamos um Config Server. As configurações são injetadas via variáveis de ambiente, gerenciadas de forma centralizada através do arquivo `.env` orquestrado pelo `docker-compose.yml`.
3. **Message Broker (Apache Kafka):** Utilizado para desacoplar a dedução de estoque da criação do pedido, garantindo altíssima vazão (throughput) e tolerância a falhas.

---

## 🧬 1. Padrões de Persistência & Auditoria

Todas as tabelas do ecossistema compartilham uma fundação robusta de governança de dados através de um struct base (`models.BaseModel`), evitando repetição de código SQL:

### ⏱️ Auditoria e Soft Delete Explícito
Diferente de frameworks ORM que usam mágica de interceptadores, a auditoria e o soft delete são gerenciados de forma explícita e performática diretamente nas queries SQL:
* **Campos Gerenciados:** `created_at`, `created_by`, `updated_at`, `updated_by`, `deleted` e `version`.
* **Soft Delete Nativo:** Os repositórios jamais executam `DELETE`. O método `DeleteById` executa:
  ```sql
  UPDATE table_name 
  SET deleted = true, updated_at = NOW(), updated_by = $1, version = version + 1 
  WHERE id = $2 AND deleted = false
  ```
* **Filtro Automático:** Todas as queries de busca (`FindById`, `FindAll`) incluem explicitamente a cláusula `WHERE deleted = false` e utilizam índices parciais otimizados do PostgreSQL.

---

## 🗄️ 2. Detalhes Técnicos dos Domínios

### 🔐 Auth Service (`ms_auth` - Porta 4001)
Centraliza o controle de usuários e as regras corporativas de segurança.
* **Segurança:** Senhas armazenadas exclusivamente via `bcrypt` (cost 12). Autenticação JWT-Assimétrica usando chaves RSA (PS256).
* **Contexto:** Middleware extraí o usuário do token e o injeta no `context.Context` do Go, permitindo que as camadas de Service e Repository saibam exatamente *quem* está executando a ação (para auditoria no `created_by`).
* **Validação:** Regras rígidas de senha (mínimo 8 caracteres) e validação de e-mail via Regex.

### 📦 Product Service (`ms_product` - Porta 4002)
Catálogo puro de itens expostos ao cliente.
* **Modelo:** Entidade simples mapeando `name` e `price`.
* **Regras de Serviço:** Suporta buscas por ID.

### 📊 Stock Service (`ms_stock` - Porta 4003)
Camada de inventário física. Isola as volumetrias e bloqueios do catálogo.
* **Controle de Concorrência:** Implementa **Optimistic Locking** nativo do PostgreSQL através da cláusula `WHERE version = $version` no `Update`, blindando contra inconsistências numéricas em concorrências massivas.
* **Comunicação Síncrona (`ProductClient`):** Durante a criação de estoque, valida se o produto existe via chamada HTTP padrão antes de efetivar o registro.
* **Consumidor Kafka:** Escuta o tópico `orders`, deduz o estoque atômico e responde o tópico `stock.check-result` para atualizar o status do pedido. Usa o padrão *System User* para auditoria de ações de sistema.

### 🛒 Order Service (`ms_order` - Porta 4004)
Orquestrador das intenções de compra do ecossistema.
* **Modelo:** Relacionamento `Order` (1) para `OrderItem` (N). O `OrderItem` armazena o `unitPrice` congelado no instante do fechamento, protegendo o histórico contra flutuações de preço.
* **Transações:** Usa um padrão de `TransactionManager` injetado no contexto para garantir que a criação do Pedido e seus Itens ocorram atomicamente.

---

## ⚙️ 3. O Fluxo de Compra Híbrido

O fechamento de uma compra utiliza uma arquitetura Híbrida (Síncrona + Assíncrona) para maximizar consistência e vazão:

### Fase 1: Validação Síncrona (HTTP Client Nativo)
1. O cliente submete a compra ao Gateway, que a encaminha ao `ms_order`.
2. O `OrderService` faz uma chamada HTTP síncrona para o `ms_stock`:
   ```http
   POST http://ms-stock:4003/v1/stock/check-availability
   ```
3. O `ms_stock` avalia o inventário em bloco. Se a quantidade solicitada for maior que a disponível, retorna um relatório detalhado (Estruturado em JSON via `DetailedApiError`), fazendo o `ms_order` abortar a transação e retornar `422 Unprocessable Entity`.

### Fase 2: Persistência e Notificação por Eventos (Kafka/Sarama)
1. Se o estoque validar, o `ms_order` salva o registro com status `PENDING` e os itens associados em uma transação DB.
2. O serviço publica um evento rico (`OrderCreatedEvent`) via **Sarama SyncProducer** (com Idempotência ativada).
3. O `ms_stock` consome a mensagem em background via **Consumer Group**, executa a dedução atômica e publica o resultado em `stock.check-result`.
4. O `ms_order` consome esse resultado e atualiza o status para `APPROVED` ou `REJECTED`.

---

## 🌐 4. Mapeamento de Endpoints Unificados no Gateway

O API Gateway centraliza todas as chamadas expostas à internet sob a porta **8080**, traduzindo o path público (`/api/...`) para o path interno dos microsserviços (`/v1/...`):

| Rota Exposta no Gateway | Alvo Interno Redirecionado | Microsserviço | Resiliência (Circuit Breaker) |
| :--- | :--- | :--- | :--- |
| `POST /api/auth` | `/v1/auth` | `ms_auth` (4001) | `msAuthCircuitBreaker` |
| `POST /api/auth/refresh-token` | `/v1/auth/refresh-token` | `ms_auth` (4001) | `msAuthCircuitBreaker` |
| `POST /api/user` | `/v1/user` | `ms_auth` (4001) | `msAuthCircuitBreaker` |
| `GET /api/product/{id}` | `/v1/product/{id}` | `ms_product` (4002) | `msProductCircuitBreaker` |
| `POST /api/stock/check-availability` | `/v1/stock/check-availability` | `ms_stock` (4003) | `msStockCircuitBreaker` |
| `POST /api/orders` | `/v1/orders` | `ms_order` (4004) | `msOrderCircuitBreaker` |

---

## 🛠️ 5. Stack Tecnológica

* **Linguagem:** Go 1.22+
* **HTTP Framework:** `go-chi/chi` (Router, Middleware)
* **Banco de Dados:** PostgreSQL 15 (Alpine)
* **Driver/Pool:** `lib/pq`, gerenciamento nativo de conexões (`database/sql`)
* **Mensageria:** Apache Kafka (Modo KRaft) + `IBM/sarama`
* **Cache:** Redis 7 (Alpine)
* **Resiliência:** `sony/gobreaker` (Circuit Breaker no Gateway)
* **Migrações:** `pressly/goose`
* **Infraestrutura:** Docker & Docker Compose (Rede bridge isolada)
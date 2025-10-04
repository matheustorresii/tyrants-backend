## Tyrants - Documentação de API (pt-BR)

Este documento descreve as APIs disponíveis no backend do Tyrants e como testá-las no Postman.

- **Base URL**: `http://localhost:8080`
- **Formato**: `application/json`
- **Campos desconhecidos**: Requisições com campos extras são rejeitadas com `400 Bad Request`.
- **Armazenamento**: SQLite (persistente em arquivo `tyrants.db`).

## Como rodar o servidor

1. No terminal, dentro do diretório do projeto, execute:

```bash
make run
```

2. O servidor ficará acessível em `http://localhost:8080`.
3. Persistência: os dados são salvos em `tyrants.db` (SQLite) no diretório raiz do projeto.
   - Para limpar os dados, pare o servidor e remova o arquivo `tyrants.db`.
   - Você pode alterar o DSN no `cmd/server/main.go` se necessário.

### Banco de dados (SQLite)

- O backend usa SQLite via driver em puro Go (sem dependências nativas), criando automaticamente o arquivo `tyrants.db` ao subir a aplicação.
- Migrações são executadas automaticamente na inicialização, criando as tabelas `users` e `news` se não existirem.
- Modo de journal: WAL (`_journal=WAL`), melhor para durabilidade e concorrência.

#### O que acontece se o servidor cair?

- Os dados ficam persistidos no arquivo `tyrants.db`. Você não perde os dados ao reiniciar o servidor.
- Transações não finalizadas são revertidas na abertura do banco. Dados já confirmados (committed) permanecem.
- Risco de perda só ocorreria por corrupção de disco/FS ou exclusão manual do arquivo.

#### Personalização

- O DSN padrão está em `cmd/server/main.go` (ex.: `file:tyrants.db?cache=shared&mode=rwc&_journal=WAL`).
- Você pode apontar para outro caminho/arquivo ou ajustar parâmetros (cache, journal, etc.).

## Como preparar o Postman

1. Abra o Postman e crie uma Collection chamada "Tyrants" (opcional, mas recomendado).
2. Para cada endpoint abaixo, crie uma nova Request dentro da Collection.
3. Configure o método HTTP, a URL e adicione o header `Content-Type: application/json`.
4. No Body, selecione "raw" e o tipo "JSON".
5. Cole o JSON de exemplo e clique em "Send".

---

## Tyrants (CRUD)

- **Coleção**: `/tyrants`
- **Item**: `/tyrants/{id}`
- **Modelo**:

```json
{
  "id": "string",
  "asset": "string",
  "nickname": "string|null",
  "evolutions": ["string", "string"],
  "attacks": [
    { "name": "string", "power": 50, "pp": 10, "attributes": ["fire", "aoe"] }
  ],
  "hp": 100,
  "attack": 20,
  "defense": 12,
  "speed": 18
}
```

Observações:
- `id` é o nome canônico do Tyrant e também sua PK.
- `asset` é uma string livre para referenciar imagens/recursos.
- `nickname` é opcional e pode ser alterado via `PUT`.
- `evolutions` é opcional; quando presente, é uma lista de nomes (ids) de outros tyrants.
- `attacks` contém golpes com `name`, `power` (int), `pp` (int) e `attributes` (lista de strings).

### Listar tyrants

- **Endpoint**: `GET /tyrants`
- **Resposta**: `200 OK` com array de tyrants

Exemplo via cURL:

```bash
curl -i http://localhost:8080/tyrants
```

### Criar tyrant

- **Endpoint**: `POST /tyrants`
- **Headers**: `Content-Type: application/json`
- **Resposta**: `201 Created`; `409 Conflict` se `id` já existir; `400 Bad Request` para payload inválido/campos extras.

Payload exemplo:

```json
{
  "id": "tumba",
  "asset": "asset-tumba",
  "nickname": "Máquina",
  "evolutions": ["tumba-evo1", "tumba-evo2"],
  "attacks": [
    {"name":"Soco Flamejante","power":60,"pp":15,"attributes":["fire"]},
    {"name":"Investida","power":40,"pp":25,"attributes":["physical"]}
  ],
  "hp": 120,
  "attack": 30,
  "defense": 20,
  "speed": 12
}
```

Exemplo via cURL:

```bash
curl -i -X POST http://localhost:8080/tyrants \
  -H 'Content-Type: application/json' \
  -d '{"id":"tumba","asset":"asset-tumba","nickname":"Máquina","evolutions":["tumba-evo1","tumba-evo2"],"attacks":[{"name":"Soco Flamejante","power":60,"pp":15,"attributes":["fire"]},{"name":"Investida","power":40,"pp":25,"attributes":["physical"]}],"hp":120,"attack":30,"defense":20,"speed":12}'
```

### Obter tyrant por ID

- **Endpoint**: `GET /tyrants/{id}`
- **Resposta**: `200 OK`; `404 Not Found` se não existir.

Exemplo via cURL:

```bash
curl -i http://localhost:8080/tyrants/tumba
```

### Atualizar tyrant

- **Endpoint**: `PUT /tyrants/{id}`
- **Headers**: `Content-Type: application/json`
- **Resposta**: `200 OK`; `404 Not Found` se não existir; `400 Bad Request` para payload inválido/campos extras.

Payload exemplo (campos opcionais `nickname`, `evolutions`, `attacks` podem ser omitidos para manter os valores atuais):

```json
{
  "asset": "asset-tumba-v2",
  "nickname": "Tumbalord",
  "evolutions": ["tumba-evo2"],
  "attacks": [
    {"name":"Soco Flamejante","power":65,"pp":15,"attributes":["fire","burn"]}
  ],
  "hp": 130,
  "attack": 35,
  "defense": 22,
  "speed": 14
}
```

Exemplo via cURL:

```bash
curl -i -X PUT http://localhost:8080/tyrants/tumba \
  -H 'Content-Type: application/json' \
  -d '{"asset":"asset-tumba-v2","nickname":"Tumbalord","evolutions":["tumba-evo2"],"attacks":[{"name":"Soco Flamejante","power":65,"pp":15,"attributes":["fire","burn"]}],"hp":130,"attack":35,"defense":22,"speed":14}'
```

### Deletar tyrant

- **Endpoint**: `DELETE /tyrants/{id}`
- **Resposta**: `204 No Content`; `404 Not Found` se não existir.

Exemplo via cURL:

```bash
curl -i -X DELETE http://localhost:8080/tyrants/tumba
```

## Criar Usuário

- **Endpoint**: `POST /users`
- **Descrição**: Cria um usuário com `id`, `name` e, opcionalmente, `admin` (bool). Se `admin=true`, o usuário não possui `tyrant`, `xp` ou `items`.
- **Headers**: `Content-Type: application/json`

### Payload (request)

```json
{
  "id": "ash-ketchum",
  "name": "Ash Ketchum",
  "admin": false
}
```

Regras de validação:
- **id**: string não vazia, escolhida pelo usuário
- **name**: string não vazia
- **admin**: booleano opcional (padrão: false)
- Campos desconhecidos não são permitidos (gera `400 Bad Request`).

### Respostas

- `201 Created` + corpo com o usuário criado

```json
{
  "id": "ash-ketchum",
  "name": "Ash Ketchum",
  "admin": false
}
```

- `409 Conflict` se já existir um usuário com o mesmo `id`.
- `400 Bad Request` se o JSON for inválido, se houver campos desconhecidos ou se faltar campo obrigatório (`id` ou `name`).

Observação: Em erros, o corpo retorna o texto do status (por exemplo, "Conflict"), não um JSON estruturado.

### Testando no Postman (passo a passo)

1. Método: `POST`
2. URL: `http://localhost:8080/users`
3. Headers: `Content-Type: application/json`
4. Body (raw, JSON): usar o payload acima
5. Clique em "Send" e verifique se o status é `201 Created` e o corpo contém o usuário.

### Teste rápido via cURL

```bash
curl -i -X POST http://localhost:8080/users \
  -H 'Content-Type: application/json' \
  -d '{"id":"ash-ketchum","name":"Ash Ketchum","admin":false}'
```

---

## Login (mockado)

- **Endpoint**: `POST /login`
- **Descrição**: Faz login com base no `id` já criado previamente.
- **Headers**: `Content-Type: application/json`

### Payload (request)

```json
{
  "id": "ash-ketchum"
}
```

Regras de validação:
- **id**: string não vazia
- Campos desconhecidos não são permitidos (gera `400 Bad Request`).

### Respostas

- `200 OK` + corpo com o usuário encontrado

```json
{
  "id": "ash-ketchum",
  "name": "Ash Ketchum"
}
```

- `404 Not Found` se o usuário não existir.
- `400 Bad Request` se o JSON for inválido, se houver campos desconhecidos ou se faltar `id`.

Observação: Em erros, o corpo retorna o texto do status (por exemplo, "Not Found"), não um JSON estruturado.

### Resposta com detalhes do usuário

O login retorna os campos adicionais do usuário e o Tyrant completo quando associado. Se `admin=true`, os campos `tyrant`, `xp` e `items` não aparecem.

```json
{
  "id": "ash-ketchum",
  "name": "Ash Ketchum",
  "admin": false,
  "tyrant": {
    "id": "tumba",
    "asset": "asset-tumba",
    "nickname": "Máquina",
    "evolutions": ["tumba-evo1"],
    "attacks": [
      {"name":"Soco Flamejante","power":60,"pp":15,"attributes":["fire"]}
    ],
    "hp": 120,
    "attack": 30,
    "defense": 20,
    "speed": 12
  },
  "xp": 123,
  "items": [
    { "name": "potion", "asset": "asset-potion" }
  ]
}
```

### Testando no Postman (passo a passo)

1. Método: `POST`
2. URL: `http://localhost:8080/login`
3. Headers: `Content-Type: application/json`
4. Body (raw, JSON): usar o payload acima
5. Clique em "Send" e verifique se o status é `200 OK` e o corpo retorna o usuário.

### Teste rápido via cURL

```bash
curl -i -X POST http://localhost:8080/login \
  -H 'Content-Type: application/json' \
  -d '{"id":"ash-ketchum"}'
```

---

## Notícias (CRUD)

- **Coleção**: `/news`
- **Item**: `/news/{id}`
- **Modelo**:

```json
{
  "id": "string",
  "image": "string",
  "title": "string",
  "content": "string",
  "date": "string",
  "category": "string|null"
}
```

Observações:
- `image` é uma string livre (não enum no backend), para permitir cadastrar novas imagens sem alterar o servidor.
- Todos os campos, exceto `category`, são obrigatórios em criação e atualização.
- Campos desconhecidos não são aceitos.

### Listar notícias

- **Endpoint**: `GET /news`
- **Resposta**: `200 OK` com array de notícias

Exemplo de teste no Postman:
1. Método: `GET`
2. URL: `http://localhost:8080/news`
3. Clique em "Send"

Exemplo via cURL:

```bash
curl -i http://localhost:8080/news
```

### Criar notícia

- **Endpoint**: `POST /news`
- **Headers**: `Content-Type: application/json`
- **Resposta**: `201 Created` com a notícia criada; `409 Conflict` se `id` já existir; `400 Bad Request` para payload inválido/campos extras.

Payload exemplo:

```json
{
  "id": "news-001",
  "image": "news-midas-signing",
  "title": "Midas assinou com a liga!",
  "content": "Detalhes sobre a assinatura...",
  "date": "2025-10-03",
  "category": "transfer"
}
```

Exemplo via cURL:

```bash
curl -i -X POST http://localhost:8080/news \
  -H 'Content-Type: application/json' \
  -d '{"id":"news-001","image":"news-midas-signing","title":"Midas assinou com a liga!","content":"Detalhes sobre a assinatura...","date":"2025-10-03","category":"transfer"}'
```

### Obter notícia por ID

- **Endpoint**: `GET /news/{id}`
- **Resposta**: `200 OK` com a notícia; `404 Not Found` se não existir.

Exemplo via cURL:

```bash
curl -i http://localhost:8080/news/news-001
```

### Atualizar notícia

- **Endpoint**: `PUT /news/{id}`
- **Headers**: `Content-Type: application/json`
- **Resposta**: `200 OK` com a notícia atualizada; `404 Not Found` se não existir; `400 Bad Request` para payload inválido/campos extras.

Payload exemplo:

```json
{
  "image": "news-rosa-handshake",
  "title": "Rosa fechou parceria",
  "content": "Detalhes da parceria...",
  "date": "2025-10-04",
  "category": "partnership"
}
```

Exemplo via cURL:

```bash
curl -i -X PUT http://localhost:8080/news/news-001 \
  -H 'Content-Type: application/json' \
  -d '{"image":"news-rosa-handshake","title":"Rosa fechou parceria","content":"Detalhes da parceria...","date":"2025-10-04","category":"partnership"}'
```

### Deletar notícia

- **Endpoint**: `DELETE /news/{id}`
- **Resposta**: `204 No Content` se excluída; `404 Not Found` se não existir.

Exemplo via cURL:

```bash
curl -i -X DELETE http://localhost:8080/news/news-001
```

## Atualizar Usuário

- Endpoint: `PUT /users/{id}`
- Descrição: Atualiza campos opcionais do usuário: `tyrant` (string, id de um tyrant), `xp` (inteiro), `items` (lista com `name`, `asset`). Campos omitidos não são alterados.
- Headers: `Content-Type: application/json`

### Payload (request)

```json
{
  "tyrant": "tumba",
  "xp": 123,
  "items": [
    { "name": "potion", "asset": "asset-potion" },
    { "name": "revive", "asset": "asset-revive" }
  ]
}
```

### Respostas

- `200 OK` + corpo com detalhes atualizados do usuário (mesmo formato do login)
- `404 Not Found` se o usuário não existir.
- `400 Bad Request` se o JSON for inválido ou contiver campos desconhecidos.

### Testando no Postman

1. Método: `PUT`
2. URL: `http://localhost:8080/users/ash-ketchum`
3. Headers: `Content-Type: application/json`
4. Body (raw, JSON): payload acima
5. Clique em "Send" e verifique `200 OK` e o corpo atualizado.

### cURL

```bash
curl -i -X PUT http://localhost:8080/users/ash-ketchum \
  -H 'Content-Type: application/json' \
  -d '{"tyrant":"tumba","xp":123,"items":[{"name":"potion","asset":"asset-potion"},{"name":"revive","asset":"asset-revive"}]}'
```

## Dicas e casos de erro

- Enviar campos extras (por exemplo, `{"id":"x","name":"y","extra":true}`) retorna `400 Bad Request`.
- Enviar `id` ou `name` vazios em `/users` retorna `400 Bad Request`.
- Criar o mesmo `id` duas vezes em `/users` retorna `409 Conflict`.
- Fazer `/login` com um `id` que não existe retorna `404 Not Found`.



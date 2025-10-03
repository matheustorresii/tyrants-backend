## Tyrants - Documentação de API (pt-BR)

Este documento descreve as APIs disponíveis no backend do Tyrants e como testá-las no Postman.

- **Base URL**: `http://localhost:8080`
- **Formato**: `application/json`
- **Campos desconhecidos**: Requisições com campos extras são rejeitadas com `400 Bad Request`.
- **Armazenamento**: Mock em memória (dados são perdidos ao reiniciar o servidor).

## Como rodar o servidor

1. No terminal, dentro do diretório do projeto, execute:

```bash
make run
```

2. O servidor ficará acessível em `http://localhost:8080`.

## Como preparar o Postman

1. Abra o Postman e crie uma Collection chamada "Tyrants" (opcional, mas recomendado).
2. Para cada endpoint abaixo, crie uma nova Request dentro da Collection.
3. Configure o método HTTP, a URL e adicione o header `Content-Type: application/json`.
4. No Body, selecione "raw" e o tipo "JSON".
5. Cole o JSON de exemplo e clique em "Send".

---

## Criar Usuário

- **Endpoint**: `POST /users`
- **Descrição**: Cria um usuário a partir de um `id` (escolhido pelo próprio usuário) e `name`.
- **Headers**: `Content-Type: application/json`

### Payload (request)

```json
{
  "id": "ash-ketchum",
  "name": "Ash Ketchum"
}
```

Regras de validação:
- **id**: string não vazia, escolhida pelo usuário
- **name**: string não vazia
- Campos desconhecidos não são permitidos (gera `400 Bad Request`).

### Respostas

- `201 Created` + corpo com o usuário criado

```json
{
  "id": "ash-ketchum",
  "name": "Ash Ketchum"
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
  -d '{"id":"ash-ketchum","name":"Ash Ketchum"}'
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

## Dicas e casos de erro

- Enviar campos extras (por exemplo, `{"id":"x","name":"y","extra":true}`) retorna `400 Bad Request`.
- Enviar `id` ou `name` vazios em `/users` retorna `400 Bad Request`.
- Criar o mesmo `id` duas vezes em `/users` retorna `409 Conflict`.
- Fazer `/login` com um `id` que não existe retorna `404 Not Found`.



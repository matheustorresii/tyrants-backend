## Protocolo WebSocket da Scene

URL: `ws://localhost:8080/scene/ws`

Mensagens são JSON. O servidor pode broadcastar atualizações em JSON para todos os clientes conectados.

### Conectar

Use um cliente WebSocket (Insomnia, Postman WebSocket, wscat, etc.).

### Mensagens do Cliente → Servidor

1) Exibir imagem (broadcast):

```json
{ "image": "https://link-ou-id-da-imagem", "fill": false }
```

2) Entrar na cena com um Tyrant (opcional `enemy`):

```json
{ "join": "tumba", "enemy": true }
```

3) Iniciar batalha (com ou sem votação):

```json
{ "battle": "tumba", "voteEnabled": true }
```

- Se `voteEnabled` for `true`, a batalha entra em fase de votação antes de iniciar turnos.

4) Executar ataque (somente o nome do ataque):

```json
{
  "attack": {
    "user": "mystelune",
    "target": "platybot",
    "attack": "Salto"
  }
}
```

5) Limpar batalha/fila (remover inimigos, manter protagonistas prontos):

```json
{ "clean": true }
```

Observações:
- O servidor valida se o ataque existe na lista de `attacks` do Tyrant atacante.
- O dano é calculado por `(atk * (random + (power * 10)) - def) / 200` com `random in [1,100]` e multiplicador 2x quando `random >= 90`. O dano mínimo é 1.
- PP: cada ataque possui `fullPP` e `currentPP` na batalha; quando `currentPP` chegar a 0, o ataque não pode ser usado até a próxima batalha.

5) Votar (apenas `enemy: false`):

```json
{ "vote": "UNTIL_DEATH", "user": "aliado1" }
```

- Valores válidos: `UNTIL_DEATH` ou `TO_PARTY`.
- Se `user` não for enviado, o servidor tenta inferir pelo socket (quando possível).

### Mensagens do Servidor → Clientes

1) Confirmação de join (apenas para quem entrou):

```json
{ "joined": "tumba", "enemy": true }
```

2) Início de votação (quando `voteEnabled = true`):

```json
{ "voting": { "UNTIL_DEATH": 0, "TO_PARTY": 0 } }
```

- O servidor vai enviar atualizações de votos a cada novo voto recebido:

```json
{ "voting": { "UNTIL_DEATH": 1, "TO_PARTY": 0 } }
```

- A votação encerra quando todos os `enemy: false` que deram `join` tiverem votado. Em caso de empate, considera `TO_PARTY`.
- Ao encerrar a votação, o servidor inicia a batalha e envia UMA mensagem contendo o resultado final da votação e a ordem de turnos:

```json
{ "battle": "", "turns": [ {"id":"...","asset":"...","enemy":false}, ... ], "voting": { "UNTIL_DEATH": 2, "TO_PARTY": 3 } }
```

3) Início de batalha e ordem de turnos (quando sem votação):

```json
{ "battle": "tumba", "turns": [ {"id":"...","asset":"...","enemy":false}, ... ] }
```

4) Atualização de estado (HP e PP dos participantes) e novos turnos:

```json
{
  "updateState": {
    "tyrants": [
      {
        "id": "mystelune",
        "fullHp": 120,
        "currentHp": 95,
        "attacks": [
          { "name": "Salto", "fullPP": 15, "currentPP": 14 },
          { "name": "Investida", "fullPP": 25, "currentPP": 25 }
        ]
      },
      {
        "id": "platybot",
        "fullHp": 110,
        "currentHp": 110,
        "attacks": [
          { "name": "Golpe", "fullPP": 20, "currentPP": 20 }
        ]
      }
    ]
  },
  "turns": [
    { "id": "aliado1",  "asset": "asset-aliado1",  "enemy": false },
    { "id": "aliado2",  "asset": "asset-aliado2",  "enemy": false },
    { "id": "inimigo2", "asset": "asset-inimigo2", "enemy": true },
    { "id": "aliado3",  "asset": "asset-aliado3",  "enemy": false },
    { "id": "inimigo1", "asset": "asset-inimigo1", "enemy": true }
  ]
}
```

4) Conclusão (vitória/derrota):

```json
{ "updateState": "WIN" }
```

ou

```json
{ "updateState": "DEFEAT" }
```

Após `WIN/DEFEAT`, somente os inimigos são removidos da fila/estado; os protagonistas permanecem conectados para próximas batalhas.

5) Confirmação de limpeza e ordem atual:

```json
{
  "clean": true,
  "turns": [
    { "id": "aliado1", "asset": "asset-aliado1", "enemy": false },
    { "id": "aliado2", "asset": "asset-aliado2", "enemy": false }
  ]
}
```

### Fluxo sugerido

1. Cada cliente envia `join` com seu `tyrant-id` (e `enemy` quando aplicável).
2. Quando todos estiverem prontos, envie `battle` para iniciar.
3. Em cada turno, o cliente do Tyrant atual envia `attack`.
4. O servidor calcula dano, emite `updateState` (com HP e PP) e `turns` (ordem atualizada).
5. Quando um lado for totalmente derrotado, emite `WIN` ou `DEFEAT` e remove apenas inimigos.

### Notas

- O hub atual é único global por servidor; se precisar de múltiplas salas, basta estender com um `roomId` e instanciar hubs por sala.
- O servidor não persiste estado da batalha; é mantido em memória e reiniciado ao reconectar.
- Para autenticação/controle de acesso, adicione um token ao header de conexão e valide no upgrade.



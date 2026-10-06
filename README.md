# leeks-mcp

Serveur MCP (stdio) en lecture seule pour l'API [LeekWars](https://leekwars.com). Six outils qui renvoient un résumé JSON compact par défaut, ou la réponse brute de l'API avec `raw: true`.

## Outils

| Outil | Paramètres | Token |
|---|---|---|
| `get_leek` | `id` | optionnel (ajoute composants et capital) |
| `get_farmer` | `id` optionnel | requis si `id` absent (habs, cristaux, inventaire) |
| `list_fights` | `leek_id`, `result` (`win`/`defeat`/`draw`), `limit` (10) | non |
| `get_fight` | `id`, `leek_id` optionnel (ne garde que l'activité de ce poireau dans les tours) | oui |
| `get_fight_logs` | `id`, `leek_id` optionnel | oui |
| `get_item` | `query` : nom (`laser`, `sun spear`) ou id | non |

Limites connues : `list_fights` ne voit que les 12 derniers combats renvoyés par `leek/get` ; les noms d'armes et de puces sont les clés anglaises de l'API.

## Build

```bash
go build -o bin/leeks-mcp ./cmd/leeks-mcp
go test ./...
```

## Token

Cherché dans l'ordre : variable `LEEKWARS_TOKEN`, fichier désigné par `LEEKWARS_KEY_FILE`, fichier `key` du répertoire courant. Sans token, seuls les outils publics répondent. Le fichier `key` est ignoré par git.

## Claude Code

```bash
claude mcp add leekwars -e LEEKWARS_KEY_FILE=/chemin/vers/key -- /chemin/vers/bin/leeks-mcp
```

Test manuel sans Claude Code :

```bash
npx -y @modelcontextprotocol/inspector --cli ./bin/leeks-mcp --method tools/list
npx -y @modelcontextprotocol/inspector --cli ./bin/leeks-mcp --method tools/call --tool-name get_leek --tool-arg id=135146
```

## Structure

- `cmd/leeks-mcp` : serveur et déclaration des outils.
- `internal/leekwars` : client HTTP (5 requêtes/s, timeout 15 s), lecture du token, cache des armes et puces.
- `internal/summary` : fonctions pures JSON brut → résumé, testées sur les fixtures de `testdata/`.

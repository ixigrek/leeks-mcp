# leeks-mcp

Serveur MCP (stdio) pour l'API [LeekWars](https://leekwars.com) : six outils de lecture et cinq outils de potager (état, lancement de combats). Les outils de lecture renvoient un résumé JSON compact par défaut, ou la réponse brute de l'API avec `raw: true`.

## Outils

| Outil | Paramètres | Token |
|---|---|---|
| `get_leek` | `id` | optionnel (ajoute composants et capital) |
| `get_farmer` | `id` optionnel | requis si `id` absent (habs, cristaux, inventaire) |
| `list_fights` | `leek_id`, `result` (`win`/`defeat`/`draw`), `limit` (10) | non |
| `get_fight` | `id`, `leek_id` optionnel (ne garde que l'activité de ce poireau dans les tours) | oui |
| `get_fight_logs` | `id`, `leek_id` optionnel | oui |
| `get_item` | `query` : nom (`laser`, `sun spear`) ou id | non |
| `get_garden` | `leek_id`, `composition_id` ou `farmer: true` (un seul) pour ajouter les adversaires proposés | oui |
| `start_solo_fight` | `leek_id`, `target_id` optionnel, `wait` | oui |
| `start_farmer_fight` | `target_id` optionnel, `wait` | oui |
| `start_team_fight` | `composition_id`, `target_id` optionnel, `wait` | oui |
| `start_boss_fight` | `boss` (id ou nom : `nasu_samurai`, `fennel_king`, `evil_pumpkin`), `participants` optionnel, `wait` | oui |

Limites connues : `list_fights` ne voit que les 12 derniers combats renvoyés par `leek/get` ; les noms d'armes et de puces sont les clés anglaises de l'API.

## Combats

Les outils `start_*` consomment un combat du potager. Sans `target_id`, l'adversaire est tiré au sort parmi ceux que propose le matchmaking (`get_garden` les liste) ; la cible tirée est renvoyée (`target_id`, `target_name`). Sans `participants`, `start_boss_fight` engage tous les poireaux de l'éleveur du token.

Par défaut la réponse est `{"fight_id", "status"}` (`status` 2 = généré, sinon en attente : `get_fight` répond « en génération » tant que le rapport n'est pas prêt). Avec `wait: true`, l'outil sonde le rapport toutes les 2 s pendant 60 s au plus et renvoie le même résumé que `get_fight`. Les erreurs de l'API (`error_fight_not_enough_fights`, `error_fight_no_such_team`…) sont renvoyées telles quelles.

Hors périmètre : lots de combats (`*-batch`, réservés à LeekWars+), défis, arène, escouades de boss à plusieurs éleveurs. Le combat d'équipe n'a pas pu être vérifié sur un vrai compte (fixture écrite à la main).

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
npx -y @modelcontextprotocol/inspector --cli ./bin/leeks-mcp -e LEEKWARS_KEY_FILE=/chemin/vers/key --method tools/call --tool-name get_garden --tool-arg leek_id=135146
npx -y @modelcontextprotocol/inspector --cli ./bin/leeks-mcp -e LEEKWARS_KEY_FILE=/chemin/vers/key --method tools/call --tool-name start_solo_fight --tool-arg leek_id=135146 --tool-arg wait=true
```

## Structure

- `cmd/leeks-mcp` : serveur et déclaration des outils.
- `internal/leekwars` : client HTTP (`Get`, `Post` JSON ; 5 requêtes/s, timeout 15 s), lecture du token, cache des armes et puces.
- `internal/summary` : fonctions pures JSON brut → résumé, testées sur les fixtures de `testdata/`.

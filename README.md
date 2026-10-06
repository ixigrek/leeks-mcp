# leeks-mcp

Serveur MCP (stdio) pour l'API [LeekWars](https://leekwars.com) : six outils de lecture, cinq outils de potager (état, lancement de combats) et quatre outils de loadouts (équipement et capital des poireaux). Les outils de lecture renvoient un résumé JSON compact par défaut, ou la réponse brute de l'API avec `raw: true`.

## Outils

| Outil | Paramètres | Token |
|---|---|---|
| `get_leek` | `id` | optionnel (ajoute composants et capital) |
| `get_farmer` | `id` optionnel | requis si `id` absent (habs, cristaux, inventaire) |
| `list_fights` | `leek_id`, `result` (`win`/`defeat`/`draw`), `limit` (10) ; historique complet (`history/get-leek-history`) | non |
| `get_fight` | `id`, `leek_id` optionnel (ne garde que l'activité de ce poireau dans les tours) | oui |
| `get_fight_logs` | `id`, `leek_id` optionnel | oui |
| `get_item` | `query` : nom (`laser`, `sun spear`) ou id (`id`/`item` d'une arme, `id` d'une puce comme dans `get_leek` ; à défaut `template` de rapport ; plusieurs objets = `ambiguous` avec `matched_by`), `kind` (`weapon`/`chip`) | non |
| `get_garden` | `leek_id`, `composition_id` et/ou `farmer: true` (combinables) pour ajouter les adversaires proposés, groupés par sélecteur | oui |
| `start_solo_fight` | `leek_id`, `target_id` optionnel, `wait` | oui |
| `start_farmer_fight` | `target_id` optionnel, `wait` | oui |
| `start_team_fight` | `composition_id`, `target_id` optionnel, `wait` | oui |
| `start_boss_fight` | `boss` (id ou nom : `nasu_samurai`, `fennel_king`, `evil_pumpkin`), `participants` optionnel, `wait` | oui |
| `run_batch` | `leek_id`, `n` (1 à 50), `type` (`solo`, `farmer`, `boss`), `boss` (type `boss`) | oui |
| `list_loadouts` | — | oui |
| `save_loadout` | `name`, `set_id` (mise à jour), `leek_id` (poireau visé, vérifié), `from_leek_id`, `weapons`, `chips` (noms), `stats` (capital par stat), `icon` | oui |
| `apply_loadout` | `set_id`, `leek_id`, `use_restat` | oui |
| `delete_loadout` | `set_id` | oui |

Limites connues : les noms d'armes et de puces sont les clés anglaises de l'API.

## Combats

Les outils `start_*` consomment un combat du potager. Sans `target_id`, l'adversaire est tiré au sort parmi ceux que propose le matchmaking (`get_garden` les liste) ; la cible tirée est renvoyée (`target_id`, `target_name`). Sans `participants`, `start_boss_fight` engage tous les poireaux de l'éleveur du token.

Par défaut la réponse est `{"fight_id", "status"}` (`status` 2 = généré, sinon en attente : `get_fight` répond « en génération » tant que le rapport n'est pas prêt). Avec `wait: true`, l'outil sonde le rapport toutes les 2 s pendant 60 s au plus et renvoie le même résumé que `get_fight`. Les erreurs de l'API (`error_fight_not_enough_fights`, `error_fight_no_such_team`…) sont renvoyées telles quelles.

`run_batch` enchaîne `n` combats au rythme d'un par seconde (adversaire retiré au sort à chaque combat, participants du boss = tous les poireaux de l'éleveur), après avoir vérifié qu'il reste assez de combats au potager, puis attend la fin de chacun et renvoie le bilan du poireau `leek_id` : `wins`, `draws`, `defeats`, `avg_turns`, `avg_life` (PV restants, 0 pour un mort) et `avg_life_wins`, `versions` (VERSION affichées au tour 1 des logs, repérées par `v<chiffre ou majuscule>…`, plusieurs jointes par ` + ` pour un combat de boss `_grp` + `_nasu`, `?` si aucune) et `defeat_ids`. Un échec de lancement arrête le lot sans perdre les combats déjà lancés ; les erreurs par combat sont listées dans `errors`.

## Loadouts

Les loadouts (ensembles d'équipement) sont le seul mécanisme de l'API pour changer l'équipement et le capital d'un poireau avec une clé API : les routes `leek/add-weapon`, `leek/remove-*` et `leek/spend-capital` exigent une session de navigateur (rôle `session` du catalogue `service/get-all`) et répondent `401 wrong_token` à une clé.

Un loadout décrit un build complet : armes, puces, composants et **capital total investi par stat** (`{"strength": 700, "tp": 255}`), l'unité que `get_leek` renvoie dans `capital_spent`. Flux type : `get_leek` → `save_loadout name=… from_leek_id=… weapons=[…] stats={strength: 760}` → `apply_loadout`. `from_leek_id` part du build actuel du poireau (composants compris, pour ne pas les perdre) ; `weapons` et `chips` remplacent la liste, `stats` se fusionne stat par stat (0 retire la stat). Le serveur ne vérifie rien à l'enregistrement : le loadout est donc vérifié localement pour le poireau visé (`leek_id`, sinon `from_leek_id`, l'un des deux requis) : emplacements d'armes et de puces, niveau requis de chaque objet, capital total du niveau, doublons.

`apply_loadout` équipe le poireau et investit le capital supplémentaire, ce qui est irréversible. Réduire le capital d'une stat exige `use_restat: true` et consomme une potion de restat (`no_restat_potion` sinon). Les erreurs de l'API (`not_enough_capital` avec `required` et `available`) sont renvoyées telles quelles ; `skipped` liste les objets non équipés.

Hors périmètre : lots de combats de l'API (`*-batch`, réservés à LeekWars+ ; `run_batch` lance les combats un par un), défis, arène, escouades de boss à plusieurs éleveurs. Le combat d'équipe n'a pas pu être vérifié sur un vrai compte (fixture écrite à la main).

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
- `internal/leekwars` : client HTTP (`Get`, `Post`, `Put`, `Delete` JSON ; 5 requêtes/s, un réessai sur 429, timeout 15 s), lecture du token, cache des armes et puces.
- `internal/summary` : fonctions pures JSON brut → résumé, testées sur les fixtures de `testdata/`.

# Claude Code · PR 2 · Persistencia y aislamiento de búsquedas recientes

**Repositorio:** `danielcanessa/mattermost`, fork público de `mattermost/mattermost`
**Responsable:** Roberto (`rdelcidw`)
**Rama:** `feature/recent-searches-persistencia`, desde `master` → PR hacia `master`
**Hito del plan:** H10 (05–06/11/2026) · Riesgos R3 (aislamiento por usuario) y R5 (límites)

La funcionalidad de búsquedas recientes ya está en `master` (PR #1, commit `7967b2f`). Este PR corrige tres hallazgos de la revisión automática que quedaron pendientes en la capa de persistencia y agrega las pruebas de límites y aislamiento.

Idioma: comentarios del issue y del PR en español; código, nombres de pruebas, commits y ramas en inglés.

---

## Reglas (no negociables)

1. **El PR va al fork del equipo, nunca a `mattermost/mattermost`.** Usá siempre `--repo danielcanessa/mattermost --base master` y mostrame el comando antes de crearlo.
2. **No menciones issues ni PR del repositorio oficial** (`mattermost#15976`, `#39343`, etc.) en commits, PR, issues ni comentarios: eso deja referencias públicas en el repo de Mattermost.
3. No hagás push, no creés issues ni PR sin mostrarme antes qué vas a hacer y recibir mi "ok".
4. Nunca `push --force` ni `rebase` sobre una rama ya subida.
5. **No modifiqués `.github/workflows/team-ci.yml`.** Si el CI falla por algo del workflow, parás y me lo reportás.
6. **En los paquetes `server/channels/api4` y `server/channels/app`, el CI solo corre pruebas cuyo nombre contiene `RecentSearch`** (`-run RecentSearch`). Toda prueba nueva en esos paquetes tiene que llamarse `TestRecentSearch…`; si no, el CI no la ejecuta.
7. **No agregués textos nuevos a `server/i18n/en.json`**: los 14 textos pendientes los agrega el PR 3 (Daniel) y tocar el mismo archivo genera conflictos. Si necesitás un error nuevo, reutilizá un id existente de `recent_search`.
8. Commits en inglés, Conventional Commits, firmados con `-s`. Identidad de git solo en este repo: `rdelcidw <290111833+rdelcidw@users.noreply.github.com>`.
9. Seguí los patrones del código existente. Si algo de este documento no coincide con el repo, parás y me lo reportás.

---

## Fase A · Verificación previa (solo lectura)

```bash
git remote -v                      # origin = danielcanessa/mattermost
git fetch origin
git log -1 --format='%h %s' origin/master    # esperado: 7967b2f Merge pull request #1 ... o posterior
git config user.name; git config user.email  # debe ser rdelcidw / noreply
gh auth status                     # cuenta activa: rdelcidw
```

Leé y resumime estos archivos antes de cambiar nada:

- `server/public/model/recent_search.go` (`PreSave`, `IsValid`, `RecentSearchMaxTermsRunes = 2048`)
- `server/channels/app/recent_search.go` (`SaveRecentSearch`)
- `server/channels/api4/recent_search.go` (handlers y `requireRecentSearchOwner`)
- `server/channels/store/sqlstore/recent_search_store.go` (`Save` con `ON CONFLICT (userid, terms, searchtype, teamid)`, `GetByUser`, `Delete`)
- `server/channels/db/migrations/postgres/000233_create_recent_search_queries.up.sql` y `server/channels/db/migrations/migrations.list`
- Las pruebas existentes: `TestRecentSearchesAPI` (api4), `TestSaveRecentSearch` y `TestDeleteRecentSearch` (app), `TestRecentSearchStore` (sqlstore)

---

## Fase B · Issue y rama (con mi ok)

```bash
gh issue create --repo danielcanessa/mattermost \
  --title "Recent searches: harden persistence and add isolation and limit tests" \
  --body "Hito H10 del plan. Riesgos R3 y R5.

Criterios de aceptación:
- [ ] El servidor ignora el id, create_at y update_at que manda el cliente al guardar.
- [ ] Las columnas id, userid y teamid son VARCHAR(26): team_id vacío se devuelve como cadena vacía.
- [ ] Guardar términos de hasta 2048 caracteres multibyte no da error 500.
- [ ] Pruebas con 0, 1, 5 y 6 búsquedas guardadas.
- [ ] Pruebas de aislamiento: un usuario no puede leer, guardar ni borrar búsquedas de otro.
- [ ] CI en verde."

git switch -c feature/recent-searches-persistencia origin/master
```

Guardá el número del issue (`<N>`).

---

## Fase C · Cambios

### C.1 El servidor controla id y fechas

Hoy `PreSave` conserva el `Id` y el `CreateAt` que manda el cliente. Un cliente puede enviar el `id` de una fila existente (de otro usuario) y provocar un conflicto de clave primaria → error 500, o sondear ids.

- En `App.SaveRecentSearch` (`server/channels/app/recent_search.go`), antes de `search.PreSave()`: poné `search.Id = ""`, `search.CreateAt = 0` y `search.UpdateAt = 0`. Se hace en la capa app para que aplique a cualquier llamador, no solo al handler HTTP.
- Pruebas (en `app` y en `api4`, con nombres `TestRecentSearch…`):
  - Guardar con un `id` y un `create_at` inventados → la respuesta trae un `id` nuevo y un `create_at` del servidor.
  - El usuario B guarda una búsqueda enviando el `id` de una búsqueda del usuario A → no hay error 500 y la fila de A queda intacta.

### C.2 Migración nueva: VARCHAR y un índice que soporte términos largos

No modifiqués la migración 000233: ya está integrada y puede haber bases locales que la corrieron. Creá la **000234**:

- `server/channels/db/migrations/postgres/000234_alter_recent_search_queries_types.up.sql`
  - `ALTER COLUMN ... TYPE VARCHAR(26)` para `id`, `userid` y `teamid`. Al convertir de `CHAR` a `VARCHAR`, PostgreSQL quita los espacios de relleno; igual agregá un `UPDATE ... SET teamid = btrim(teamid)` por seguridad.
  - Reemplazá el índice único `idx_recentsearchqueries_userid_terms_searchtype_teamid`: hoy incluye `terms` completo (hasta 2048 caracteres multibyte, es decir hasta ~8 KB), y un índice btree de PostgreSQL falla con filas de más de ~2,7 KB. Recrealo sobre `md5(terms)`: `(userid, md5(terms), searchtype, teamid)`.
- `...000234_alter_recent_search_queries_types.down.sql`: revierte las dos cosas.
- Agregá los dos archivos a `server/channels/db/migrations/migrations.list`, después de la 000233, siguiendo el formato existente.
- En `SqlRecentSearchStore.Save`, el `ON CONFLICT` tiene que coincidir con el índice nuevo: `ON CONFLICT (userid, (md5(terms)), searchtype, teamid)` (en PostgreSQL, una expresión dentro del objetivo del conflicto va entre paréntesis).
- Si encontrás una forma más simple que respete el límite de 2048 caracteres, proponémela antes de implementarla. No bajés el límite sin consultarme.

Pruebas de store (en el patrón de `TestRecentSearchStore`):
- Guardar con `TeamId: ""` → al leerla, `TeamId == ""` (no 26 espacios).
- Guardar un término de 2048 caracteres multibyte (por ejemplo `strings.Repeat("ñ", 2048)`) → sin error.
- Guardar dos veces ese mismo término largo → queda una sola fila (el upsert funciona con el índice nuevo).

### C.3 Pruebas de límites (R5)

Extendé las pruebas de store (o las existentes como `testRecentSearchRetainFive`) para cubrir:
- 0 búsquedas → `GetByUser` devuelve lista vacía, no error.
- 1 búsqueda → se devuelve.
- 5 búsquedas → se conservan las 5.
- 6 búsquedas → se conservan 5 y se elimina la más antigua.

Para que el orden sea determinista, fijá `UpdateAt` explícitamente en las pruebas de store en lugar de depender de `GetMillis()`.

### C.4 Pruebas de aislamiento (R3), en `api4`

Con dos usuarios de prueba (A y B), en pruebas `TestRecentSearch…`:
- B hace GET, POST y DELETE sobre la ruta de A (`/api/v4/users/{A}/recent_searches...`) → 403 en los tres.
- B hace DELETE en su propia ruta con el `id` de una búsqueda de A → 404, y la búsqueda de A sigue existiendo.
- Un administrador tampoco puede leer el historial de otro usuario (el handler lo bloquea con `requireRecentSearchOwner`); dejalo cubierto por una prueba.

---

## Fase D · Verificación local

Las pruebas de store, app y api4 necesitan la base de datos de desarrollo de Mattermost (Docker):

```bash
cd server
make start-docker
go test -count=1 ./channels/store/sqlstore/ -run RecentSearch
go test -count=1 ./channels/app/ -run RecentSearch
go test -count=1 ./channels/api4/ -run RecentSearch
gofmt -l $(git diff --name-only --relative origin/master...HEAD -- '*.go')
go vet ./channels/app/ ./channels/api4/ ./channels/store/... ./public/model/
make stop-docker
```

Además, probá la migración en limpio y hacia atrás (up → down → up) y reportame cómo lo hiciste. Mostrame la salida y `git diff --stat origin/master...HEAD`.

---

## Fase E · Commit, push y PR (con mi ok)

Commits sugeridos (con `-s`):
1. `fix: ignore client-supplied id and timestamps when saving recent searches`
2. `fix: use varchar columns and hashed unique index for recent searches`
3. `test: cover recent search limits and per-user isolation`

```bash
git push -u origin feature/recent-searches-persistencia
gh pr create --repo danielcanessa/mattermost --base master --head feature/recent-searches-persistencia \
  --title "fix: harden recent search persistence and add isolation tests" \
  --body-file <archivo temporal>
```

Cuerpo según `.github/PULL_REQUEST_TEMPLATE.md`: Summary (qué corrige y qué prueba), Ticket Link `Closes #<N>`, Release Note `NONE`, y al final «Cualquier integrante del equipo puede revisarlo y aprobarlo». Sin revisor asignado.

Después:
1. `gh pr view --repo danielcanessa/mattermost --json url,baseRefName,closingIssuesReferences` para confirmar base `master` y el enlace al issue.
2. `gh run watch --repo danielcanessa/mattermost` y reportame cada job. Revisá en el log del job server que tus pruebas nuevas sí aparecen ejecutadas.

---

## Reporte final

| Elemento | Valor |
| --- | --- |
| Issue | #N (URL) |
| PR | #M (URL) |
| Corrida de Team CI | URL y resultado por job |
| Pruebas nuevas | Nombre de cada una y paquete |
| Desviaciones | Lista |

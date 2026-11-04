# Claude Code · PR 3 · Orden, regresión y ajustes de código de búsquedas recientes

**Repositorio:** `danielcanessa/mattermost`, fork público de `mattermost/mattermost`
**Responsable:** Daniel
**Rama:** `feature/recent-searches-orden`, desde `master` → PR hacia `master`
**Hitos del plan:** H11 (07–08/11/2026) y H18 (ajustes de código) · Riesgo R2 (regresiones en la búsqueda existente)

La funcionalidad de búsquedas recientes ya está en `master` (PR #1, commit `7967b2f`). Este PR agrega las pruebas de orden por recencia y de regresión, y cierra dos ajustes que quedaron pendientes de la revisión automática: los textos de error faltantes y la regeneración de las capas del store.

Idioma: comentarios del issue y del PR en español; código, nombres de pruebas, commits y ramas en inglés.

> **Dependencia con el PR #7 de CI (https://github.com/danielcanessa/mattermost/pull/7).** Las pruebas del store se escriben en `channels/store/storetest`, pero solo se ejecutan desde `channels/store/sqlstore`. El PR #7 hace que, si cambia `storetest`, el CI también corra `sqlstore`, y que el log muestre por nombre cada prueba que contiene `RecentSearch`. Antes de abrir este PR, revisá con `gh pr view 7 --repo danielcanessa/mattermost --json state` si ya está integrado (`MERGED`):
> - Si ya está: traé `master` a tu rama con `git merge origin/master` y, cuando corra el CI, confirmá en el log del job `server` (paso `go test`) que aparecen por nombre tus subpruebas, por ejemplo `--- PASS: TestRecentSearchStore/PostgreSQL/<tu subprueba>`.
> - Si todavía no está: abrí el PR igual, pero anotá en la descripción que las pruebas del store se verificaron solo localmente, hasta que se integre el PR #7.
>
> **Si el PR #5 (persistencia, https://github.com/danielcanessa/mattermost/pull/5) ya está integrado,** traé `master` antes de escribir tus pruebas de store. Ese PR cambió `storetest/recent_search_store.go`: reemplazó `RetainFive` por `Limits` y agregó `EmptyTeamId` y `LongMultibyteTerms`. Así evitás conflictos en ese archivo. Si no está integrado y después aparece un conflicto, resolvelo con merge, dejando las pruebas de los dos lados.

---

## Reglas (no negociables)

1. **El PR va al fork del equipo, nunca a `mattermost/mattermost`.** Usá siempre `--repo danielcanessa/mattermost --base master` y mostrame el comando antes de crearlo.
2. **No menciones issues ni PR del repositorio oficial** (`mattermost#15976`, `#39343`, etc.) en commits, PR, issues ni comentarios: eso deja referencias públicas en el repo de Mattermost.
2b. **Nunca escribás `#<número>` para algo que no sea un issue o PR de este fork** (por ejemplo, "corrida #10"). Si el número no existe en el fork, GitHub lo enlaza al repo oficial `mattermost/mattermost` y deja una referencia pública allá. Para corridas del CI escribí "corrida 10" o pegá la URL completa. Antes de crear un issue, PR o comentario, revisá el texto con `grep -nE '#[0-9]+'` y confirmá que cada número es un issue o PR existente de `danielcanessa/mattermost`.
3. No hagás push, no creés issues ni PR sin mostrarme antes qué vas a hacer y recibir mi "ok".
4. Nunca `push --force` ni `rebase` sobre una rama ya subida.
5. **No modifiqués `.github/workflows/team-ci.yml`.** Si el CI falla por algo del workflow, parás y me lo reportás.
6. **En los paquetes `server/channels/api4` y `server/channels/app`, el CI solo corre pruebas cuyo nombre contiene `RecentSearch`** (`-run RecentSearch`). Toda prueba nueva en esos paquetes tiene que llamarse `TestRecentSearch…`; si no, el CI no la ejecuta.
7. Este PR es el único que toca `server/i18n/en.json` entre los PR de pruebas. Si al momento de abrirlo ya se integró otro PR que agregó ids de `recent_search`, traé `master` con merge antes de agregar los tuyos.
8. Commits en inglés, Conventional Commits, firmados con `-s`, con tu propia identidad de GitHub (preferiblemente el correo `noreply` de tu cuenta). Revisá `git config user.name` y `git config user.email` antes del primer commit.
9. Seguí los patrones del código existente y los comandos del `Makefile` de Mattermost. Si algo de este documento no coincide con el repo, parás y me lo reportás.

---

## Fase A · Verificación previa (solo lectura)

```bash
git remote -v                      # origin = danielcanessa/mattermost
git fetch origin
git log -1 --format='%h %s' origin/master
git config user.name; git config user.email
gh auth status
```

Leé y resumime antes de cambiar nada:

- `server/channels/store/sqlstore/recent_search_store.go`: `GetByUser` ordena por `updateat DESC`; `Save` hace upsert y al repetir una búsqueda solo actualiza `updateat`.
- `webapp/channels/src/packages/mattermost-redux/src/reducers/entities/search.ts` y su prueba `search.test.ts`.
- `webapp/channels/src/packages/mattermost-redux/src/selectors/entities/search.ts`.
- `webapp/channels/src/components/new_search/recent_searches.tsx` y `recent_searches.test.tsx`.
- `server/channels/store/store.go`, alrededor de la línea 1502: el comentario temporal que dice que falta regenerar las capas con `make store-layers`.
- Los objetivos `store-layers`, `i18n-extract` e `i18n-check` en `server/Makefile`.

---

## Fase B · Issue y rama (con mi ok)

```bash
gh issue create --repo danielcanessa/mattermost \
  --title "Recent searches: ordering and regression tests, i18n strings and store layers" \
  --body "Hitos H11 y H18 del plan. Riesgo R2.

Criterios de aceptación:
- [ ] Pruebas de orden: de la más reciente a la más antigua, y una búsqueda repetida sube al primer lugar.
- [ ] Pruebas de interfaz: el panel muestra máximo 5, en el orden recibido.
- [ ] Regresión: la suite existente de la búsqueda sigue en verde (local y CI).
- [ ] Los 14 textos de error de recent_search existen en server/i18n/en.json.
- [ ] Capas del store regeneradas con make store-layers y sin el comentario temporal en store.go.
- [ ] CI en verde."

git switch -c feature/recent-searches-orden origin/master
```

Guardá el número del issue (`<N>`).

---

## Fase C · Cambios

### C.1 Pruebas de orden (servidor)

En las pruebas de store (patrón de `TestRecentSearchStore`):
- Guardar A, B y C con `UpdateAt` crecientes → `GetByUser` devuelve C, B, A.
- Volver a guardar A con un `UpdateAt` mayor → queda una sola fila de A (upsert) y ahora el orden es A, C, B.

Fijá `UpdateAt` explícitamente en estas pruebas: si dependen de `GetMillis()`, dos guardados en el mismo milisegundo hacen que el orden sea aleatorio.

### C.2 Pruebas de orden (webapp)

- Reducer: `receivedRecentSearch` con una búsqueda nueva la pone primero; con una búsqueda que ya existe (mismo `id`) no la duplica y la sube al primer lugar. `removedRecentSearch` la quita. Extendé `search.test.ts`.
- Componente `recent_searches.tsx`: con 6 elementos en el estado, se muestran solo 5 y en el orden recibido. Extendé `recent_searches.test.tsx` usando `renderWithContext`, como las demás pruebas de `new_search`.

Si el reducer no se comporta como describe C.2, no lo cambiés por tu cuenta: reportame qué hace hoy y proponé el ajuste.

### C.3 Regresión de la búsqueda existente (R2)

El CI corre la suite completa de `webapp/channels/src/components/new_search`, pero en el servidor solo las pruebas `RecentSearch` de `api4` y `app` (decisión del equipo por el tiempo del job). Para cubrir ese hueco con evidencia:
- Buscá en `server/channels/app` y `server/channels/api4` las pruebas existentes de búsqueda de mensajes y archivos (por ejemplo, las que contienen `SearchPosts` o `SearchFiles` en el nombre) y corrélas localmente.
- Copiá en la descripción del PR el comando exacto y el resumen del resultado. Esa es la evidencia de regresión del servidor para el documento del curso.

### C.4 Textos de error (`server/i18n/en.json`)

Estos 14 ids se usan en el código y no existen en `en.json`:

```
api.recent_search.permissions.app_error
api.recent_search.save.invalid.app_error
app.recent_search.delete.app_error
app.recent_search.delete.not_found.app_error
app.recent_search.get.app_error
app.recent_search.save.app_error
model.recent_search.is_valid.create_at.app_error
model.recent_search.is_valid.id.app_error
model.recent_search.is_valid.search_type.app_error
model.recent_search.is_valid.team_id.app_error
model.recent_search.is_valid.terms.app_error
model.recent_search.is_valid.terms_length.app_error
model.recent_search.is_valid.update_at.app_error
model.recent_search.is_valid.user_id.app_error
```

- Usá el mecanismo de Mattermost: `make i18n-extract` desde `server/`, completá en inglés los textos que queden vacíos y verificá con `make i18n-check`. Si el `extract` agrega o borra ids que no son de `recent_search`, no los incluyás en el commit: reportámelo.
- Textos breves en inglés, con el mismo estilo de los ids vecinos (por ejemplo: "You do not have permission to access this user's recent searches.", "Unable to save the recent search.", "Invalid search terms.").
- Confirmá con `grep` que los 14 ids quedaron en `en.json`.

### C.5 Capas del store

- Desde `server/`, corré `make store-layers`.
- El diff debe limitarse a agregar `RecentSearch` en las capas generadas. Si cambia código de otras entidades, parás y me lo reportás.
- Borrá el comentario temporal de `server/channels/store/store.go` (cerca de la línea 1502) que dice que falta regenerar las capas.
- `go build ./...` desde `server/` tiene que compilar.

---

## Fase D · Verificación local

```bash
cd server
make start-docker
go test -count=1 ./channels/store/sqlstore/ -run RecentSearch
go test -count=1 ./channels/app/ -run RecentSearch
go test -count=1 ./channels/api4/ -run RecentSearch
# + las pruebas de búsqueda existentes de C.3
make i18n-check
go vet ./channels/store/... ./channels/app/ ./channels/api4/
make stop-docker

cd ../webapp/channels
npm run check
npm run check-types
npx jest src/components/new_search src/packages/mattermost-redux/src/reducers/entities/search.test.ts
```

Mostrame la salida y `git diff --stat origin/master...HEAD`.

---

## Fase E · Commit, push y PR (con mi ok)

Commits sugeridos (con `-s`):
1. `test: cover recent search ordering in store and webapp`
2. `chore: add missing recent search i18n strings`
3. `chore: regenerate store layers for recent searches`

```bash
git push -u origin feature/recent-searches-orden
gh pr create --repo danielcanessa/mattermost --base master --head feature/recent-searches-orden \
  --title "test: recent search ordering, i18n strings and store layers" \
  --body-file <archivo temporal>
```

Cuerpo según `.github/PULL_REQUEST_TEMPLATE.md`: Summary (incluí la evidencia de regresión de C.3), Ticket Link `Closes #<N>`, Release Note `NONE`, y al final «Cualquier integrante del equipo puede revisarlo y aprobarlo». Sin revisor asignado.

Después:
1. `gh pr view --repo danielcanessa/mattermost --json url,baseRefName,closingIssuesReferences`
2. `gh run watch --repo danielcanessa/mattermost` y reportame cada job.

---

## Reporte final

| Elemento | Valor |
| --- | --- |
| Issue | #N (URL) |
| PR | #M (URL) |
| Corrida de Team CI | URL y resultado por job |
| Pruebas nuevas | Nombre y paquete de cada una |
| Evidencia de regresión del servidor | Comando y resultado |
| Desviaciones | Lista |

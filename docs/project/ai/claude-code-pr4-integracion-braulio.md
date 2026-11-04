# Claude Code · PR 4 · Integración interfaz–servidor de búsquedas recientes

**Repositorio:** `danielcanessa/mattermost`, fork público de `mattermost/mattermost`
**Responsable:** Braulio
**Rama:** `feature/recent-searches-integracion`, desde `master` → PR hacia `master`
**Hito del plan:** H13 (11/11/2026) · Riesgo R4 (contrato entre interfaz y servidor) y R3 (aislamiento de eventos)

La funcionalidad de búsquedas recientes ya está en `master` (PR #1, commit `7967b2f`). Este PR verifica que la interfaz y el servidor hablen el mismo idioma: la API documentada, el cliente y los eventos en tiempo real por websocket.

Idioma: comentarios del issue y del PR en español; código, nombres de pruebas, commits y ramas en inglés.

> **Estado de los otros PR del equipo.** Revisá con `gh pr list --repo danielcanessa/mattermost --state all` cuáles ya se integraron, y traé `master` a tu rama (`git merge origin/master`, nunca rebase) antes de abrir el PR:
> - **PR #5, persistencia (https://github.com/danielcanessa/mattermost/pull/5).** Agrega `TestRecentSearchServerOwnedFields` y `TestRecentSearchIsolation` en `api4/recent_search_test.go`, el mismo archivo donde vas a agregar tu prueba. Si al traer `master` hay conflicto, dejá las pruebas de los dos lados. También corrige el `team_id` vacío («todos los equipos»), y eso se relaciona con C.5.
> - **PR #7, CI (https://github.com/danielcanessa/mattermost/pull/7).** Hace que el log del job `server` muestre por nombre cada prueba `RecentSearch`. Con él integrado, podés confirmar en el log que `TestRecentSearchWebSocketEvents` corrió. Sin él, el log solo muestra `ok <paquete>`; en ese caso, anotá en el PR que la prueba se verificó localmente con `-v`.

---

## Reglas (no negociables)

1. **El PR va al fork del equipo, nunca a `mattermost/mattermost`.** Usá siempre `--repo danielcanessa/mattermost --base master` y mostrame el comando antes de crearlo.
2. **No menciones issues ni PR del repositorio oficial** (`mattermost#15976`, `#39343`, etc.) en commits, PR, issues ni comentarios: eso deja referencias públicas en el repo de Mattermost.
2b. **Nunca escribás `#<número>` para algo que no sea un issue o PR de este fork** (por ejemplo, "corrida #10"). Si el número no existe en el fork, GitHub lo enlaza al repo oficial `mattermost/mattermost` y deja una referencia pública allá. Para corridas del CI escribí "corrida 10" o pegá la URL completa. Antes de crear un issue, PR o comentario, revisá el texto con `grep -nE '#[0-9]+'` y confirmá que cada número es un issue o PR existente de `danielcanessa/mattermost`.
3. No hagás push, no creés issues ni PR sin mostrarme antes qué vas a hacer y recibir mi "ok".
4. Nunca `push --force` ni `rebase` sobre una rama ya subida.
5. **No modifiqués `.github/workflows/team-ci.yml`.** Si el CI falla por algo del workflow, parás y me lo reportás.
6. **En los paquetes `server/channels/api4` y `server/channels/app`, el CI solo corre pruebas cuyo nombre contiene `RecentSearch`** (`-run RecentSearch`). Toda prueba nueva en esos paquetes tiene que llamarse `TestRecentSearch…`; si no, el CI no la ejecuta.
7. **No agregués textos a `server/i18n/en.json`**: los agrega el PR 3. Si necesitás un error nuevo, reutilizá un id existente de `recent_search`.
8. Commits en inglés, Conventional Commits, firmados con `-s`, con tu propia identidad de GitHub (preferiblemente el correo `noreply` de tu cuenta). Revisá `git config user.name` y `git config user.email` antes del primer commit.
9. Seguí los patrones del código existente. Si algo de este documento no coincide con el repo, parás y me lo reportás.

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

- **API documentada:** `api/v4/source/recent_searches.yaml`.
- **Servidor:** `server/channels/api4/recent_search.go` (rutas GET/POST en `/api/v4/users/{user_id}/recent_searches` y DELETE en `.../recent_searches/{search_id}`; `requireRecentSearchOwner`) y `App.publishRecentSearchEvent` en `server/channels/app/recent_search.go` (evento `recent_search_updated` con `action` = `save` + `item`, o `delete` + `id`, enviado solo al usuario dueño).
- **Cliente:** `getRecentSearches`, `saveRecentSearch` y `deleteRecentSearch` en `webapp/platform/client/src/client4.ts`, y su prueba en `client4.test.ts`.
- **Webapp:** `handleRecentSearchUpdatedEvent` en `webapp/channels/src/actions/websocket_actions.ts` (despacha `receivedRecentSearch` o `removedRecentSearch`) y su prueba en `websocket_actions.test.ts`.
- **Prueba end-to-end:** `e2e-tests/playwright/specs/functional/channels/search/recent_searches.spec.ts`.

---

## Fase B · Issue y rama (con mi ok)

```bash
gh issue create --repo danielcanessa/mattermost \
  --title "Recent searches: frontend-backend integration tests" \
  --body "Hito H13 del plan. Riesgos R4 y R3.

Criterios de aceptación:
- [ ] La documentación de la API coincide con lo que hace el servidor (rutas, códigos de respuesta y campos).
- [ ] Prueba de integración: el evento recent_search_updated llega al dueño al guardar y al borrar, y no llega a otro usuario.
- [ ] Prueba de la webapp: el evento actualiza el estado (save agrega, delete quita).
- [ ] Prueba end-to-end ejecutada localmente, con evidencia en el PR.
- [ ] CI en verde."

git switch -c feature/recent-searches-integracion origin/master
```

Guardá el número del issue (`<N>`).

---

## Fase C · Cambios

### C.1 Contrato de la API

Compará `api/v4/source/recent_searches.yaml` con los handlers reales:
- Rutas, métodos y parámetros (`user_id`, `search_id`).
- Códigos de respuesta: 200, 400 (cuerpo inválido), 401, 403 (otro usuario), 404 (búsqueda inexistente).
- Campos del cuerpo y de la respuesta: `id`, `user_id`, `terms`, `search_type` (`messages` o `files`), `team_id`, `create_at`, `update_at`; y la envoltura `items` del GET.

Corregí la documentación donde no coincida. Si lo que está mal es el código y no la documentación, no lo cambiés: reportámelo.

### C.2 Integración de eventos por websocket (servidor), en `api4`

Prueba nueva `TestRecentSearchWebSocketEvents` (el nombre tiene que contener `RecentSearch`), usando los helpers de websocket que ya usan otras pruebas de `api4` (buscá cómo se conecta un cliente websocket y cómo se espera un evento por tipo):
- El usuario A guarda una búsqueda → el websocket de A recibe `recent_search_updated` con `action: save` y el `item` guardado.
- El usuario A borra esa búsqueda → el websocket de A recibe `recent_search_updated` con `action: delete` y el `id`.
- El usuario B, conectado al mismo tiempo, **no** recibe ninguno de esos eventos (aislamiento, R3). Usá un tiempo de espera corto y explícito para afirmar que no llegó nada.

### C.3 Webapp

- Extendé `websocket_actions.test.ts`: un mensaje `recent_search_updated` con `action: save` despacha `receivedRecentSearch` con el item; con `action: delete` despacha `removedRecentSearch` con el id; un `action` desconocido no rompe nada.
- Revisá que `client4.test.ts` cubra las tres rutas con método y URL correctos; completá lo que falte.

### C.4 Prueba end-to-end (local, no corre en el CI)

El CI no ejecuta Playwright. Corré localmente la prueba `recent_searches.spec.ts` contra un servidor de desarrollo (`make run-server` en `server/` y la webapp según `e2e-tests/playwright/README` o la documentación del repo):
- Si pasa, adjuntá al PR el resultado y capturas o el video que genera Playwright.
- Si el entorno es demasiado pesado para tu máquina, no lo forcés: reportame qué falta y lo resolvemos aparte.

### C.5 Dependencia con el PR #5

El PR #5 (persistencia) corrige que `team_id` vacío («todos los equipos») vuelva como 26 espacios. Si el PR #5 ya está integrado cuando hagás la prueba end-to-end, verificá que reutilizar una búsqueda hecha en «todos los equipos» funciona. Si todavía no está integrado, anotalo en el PR como pendiente.

---

## Fase D · Verificación local

```bash
cd server
make start-docker
go test -count=1 -v ./channels/api4/ -run RecentSearch
go vet ./channels/api4/
make stop-docker

cd ../webapp/channels
npm run check
npm run check-types
npx jest src/actions/websocket_actions.test.ts

cd ../platform/client
npx jest src/client4.test.ts
```

Mostrame la salida y `git diff --stat origin/master...HEAD`.

---

## Fase E · Commit, push y PR (con mi ok)

Commits sugeridos (con `-s`):
1. `docs: align recent searches API reference with handlers`
2. `test: cover recent search websocket events and isolation`
3. `test: cover recent search websocket handling in webapp`

```bash
git push -u origin feature/recent-searches-integracion
gh pr create --repo danielcanessa/mattermost --base master --head feature/recent-searches-integracion \
  --title "test: recent searches frontend-backend integration" \
  --body-file <archivo temporal>
```

Cuerpo según `.github/PULL_REQUEST_TEMPLATE.md`: Summary (incluí el resultado de la prueba end-to-end o por qué quedó pendiente), Ticket Link `Closes #<N>`, Screenshots (las de Playwright, si las hay), Release Note `NONE`, y al final «Cualquier integrante del equipo puede revisarlo y aprobarlo». Sin revisor asignado.

Después:
1. `gh pr view --repo danielcanessa/mattermost --json url,baseRefName,closingIssuesReferences`
2. `gh run watch --repo danielcanessa/mattermost` y reportame cada job. Revisá en el log del job server que `TestRecentSearchWebSocketEvents` sí aparece ejecutada.

---

## Reporte final

| Elemento | Valor |
| --- | --- |
| Issue | #N (URL) |
| PR | #M (URL) |
| Corrida de Team CI | URL y resultado por job |
| Diferencias encontradas en el contrato | Lista |
| Prueba end-to-end | Resultado o motivo de pendiente |
| Desviaciones | Lista |

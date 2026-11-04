# Claude Code · Ajustes de texto del PR #10 (Braulio) · H13

**Repositorio:** `danielcanessa/mattermost`
**PR:** #10, https://github.com/danielcanessa/mattermost/pull/10. Autor: `brauliore`, rama `feature/recent-searches-tests-br`.
**Lo hace:** Roberto (`rdelcidw`), como colaborador del repo.

El código del PR #10 está bien y el CI pasó (corrida 17). Solo hay que corregir textos: crear el issue de H13, cambiar el título del PR y reemplazar su descripción. **No se toca código, ni la rama, ni se hacen commits.**

## Reglas
1. Todo con `--repo danielcanessa/mattermost`. Nada en `mattermost/mattermost`.
2. **Nunca escribás `#<número>` si no es un issue o PR existente del fork.** Para corridas del CI usá «corrida 17» o la URL completa. Antes de crear o editar cualquier texto, listá todos los `#N` que contiene y confirmá que cada uno existe en el fork.
3. Pedime ok antes de crear el issue y antes de editar el PR. Mostrame el comando y el texto exacto.
4. No apruebes ni hagás merge. Eso lo hago yo desde la web.
5. Verificá con `gh auth status` que la cuenta activa sea `rdelcidw`.

## Fase A · Verificación (solo lectura)
```bash
gh auth status
gh pr view 10 --repo danielcanessa/mattermost --json number,title,state,headRefName,baseRefName,closingIssuesReferences,mergeable,body
gh issue list --repo danielcanessa/mattermost --state all --limit 20
```
Confirmá que el PR #10 sigue abierto, con base `master`, sin conflictos, y que su descripción todavía dice `Closes #8`. Si algo cambió (por ejemplo, Braulio ya lo corrigió), parás y me lo reportás.

## Fase B · Crear el issue (con mi ok)
```bash
gh issue create --repo danielcanessa/mattermost \
  --title "Recent searches: frontend-backend integration tests" \
  --body-file <archivo temporal>
```
Contenido del archivo:
```
Hito H13 del plan. Riesgos R4 (contrato entre interfaz y servidor) y R3 (aislamiento de eventos).

Criterios de aceptación:
- [ ] La documentación de la API coincide con lo que hace el servidor (rutas, códigos de respuesta y campos).
- [ ] El evento recent_search_updated llega al dueño al guardar y al borrar, y no llega a otro usuario.
- [ ] La webapp actualiza el estado con el evento (save agrega, delete quita).
- [ ] Prueba end-to-end ejecutada localmente, con evidencia en el PR.
- [ ] CI en verde.
```
Guardá el número que le asigna GitHub (`<N>`).

## Fase C · Editar el PR #10 (con mi ok)
```bash
gh pr edit 10 --repo danielcanessa/mattermost \
  --title "test: recent searches frontend-backend integration" \
  --body-file <archivo temporal>
```
Contenido del archivo (reemplazá `<N>` por el número del issue de la Fase B):

````
#### Summary
Pruebas de integración entre la interfaz y el servidor para búsquedas recientes. Hito H13, riesgos R4 y R3.

- **Contrato de la API** (`api/v4/source/recent_searches.yaml`): se documenta la respuesta 400 en GET y DELETE, y se aclara que un id ajeno devuelve 404 y que usar el user_id de otro usuario devuelve 403.
- **Eventos por websocket** (`TestRecentSearchWebSocketEvents`, api4): al guardar, el dueño recibe `recent_search_updated` con `action: save` y el item guardado; al borrar, recibe `action: delete` con el id. Un segundo usuario conectado no recibe ninguno de los dos eventos.
- **Webapp** (`websocket_actions.test.ts`): el evento `save` despacha `RECEIVED_RECENT_SEARCH` con el item, `delete` despacha `REMOVED_RECENT_SEARCH` con el id, y una acción desconocida no despacha nada ni rompe.

Validación:
- `go test -count=1 -v ./channels/api4/ -run RecentSearch`
- `npx jest src/actions/websocket_actions.test.ts`
- `npm run check` y `npm run check-types`
- Team CI: corrida 17 (https://github.com/danielcanessa/mattermost/actions/runs/41566556314), en verde. En el log del job server aparece `--- PASS: TestRecentSearchWebSocketEvents`.

Prueba end-to-end (Playwright, `recent_searches.spec.ts`): pendiente. No se ejecutó en este PR; queda registrada para el hito de pruebas de integración del entregable final.

#### Ticket Link
Closes #<N>

#### Release Note
```release-note
NONE
```

Cualquier integrante del equipo puede revisarlo y aprobarlo.
````

## Fase D · Verificación
```bash
gh pr view 10 --repo danielcanessa/mattermost --json title,body,closingIssuesReferences
```
- El título es `test: recent searches frontend-backend integration`.
- El cuerpo publicado es idéntico al archivo, y el único `#N` que contiene es el del issue nuevo.
- `closingIssuesReferences` muestra el issue nuevo y no el #8. Si sale vacío, esperá un minuto y volvé a consultar: GitHub tarda en indexarlo.

## Reporte final
| Elemento | Valor |
| --- | --- |
| Issue nuevo | número y URL |
| PR #10 | título nuevo y closingIssuesReferences |
| `#N` en los textos | lista y confirmación de que todos existen en el fork |

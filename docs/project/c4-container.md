# Diagrama C4 · Nivel 2 (contenedores) de la mejora de búsquedas recientes

Complementa el [diagrama de contexto (nivel 1)](c4-context.md) con la vista de contenedores limitada a la funcionalidad implementada en este fork: la webapp consume tres rutas REST del servidor y recibe el evento WebSocket `recent_search_updated`; el servidor persiste el historial por usuario en PostgreSQL.

![C4 nivel 2: contenedores de búsquedas recientes](c4-container.png)

Versión en Mermaid (misma información, se renderiza en GitHub):

```mermaid
flowchart LR
  classDef person fill:#08427b,stroke:#052e56,color:#fff
  classDef container fill:#438dd5,stroke:#2e6295,color:#fff

  miembro["<b>Miembro del equipo</b><br/>[Persona]<br/>Busca mensajes y archivos; reutiliza o elimina búsquedas recientes"]:::person

  subgraph mm ["Mattermost (fork del equipo)"]
    direction LR
    webapp["<b>Webapp</b><br/>[React, Redux, TypeScript]<br/>Panel new_search: hasta 5 búsquedas recientes, reutilizar y eliminar.<br/>Cliente REST (client4) y manejador del evento WebSocket"]:::container
    server["<b>Servidor</b><br/>[Go]<br/>api4: GET/POST/DELETE /users/{id}/recent_searches<br/>app: Save, Get, Delete y evento recent_search_updated<br/>store: SqlRecentSearchStore (upsert, orden por updateat, límite 5)"]:::container
    db[("<b>PostgreSQL</b><br/>Tabla recentsearchqueries<br/>índice único (userid, md5(terms), searchtype, teamid)<br/>migraciones 000233 y 000234")]:::container
  end

  miembro -- "Usa [HTTPS]" --> webapp
  webapp -- "GET / POST / DELETE recent_searches [REST, JSON]" --> server
  server -. "recent_search_updated: save | delete, solo al dueño [WebSocket]" .-> webapp
  server -- "Lee y escribe el historial del usuario [SQL]" --> db
```

## Elementos

| Elemento | Tipo | Tecnología | Responsabilidad en la mejora |
|---|---|---|---|
| Webapp | Contenedor | React, Redux, TypeScript | `components/new_search/recent_searches.tsx` muestra hasta cinco búsquedas, permite reutilizarlas (`handleRerun`) y eliminarlas; `client4.ts` expone `getRecentSearches`, `saveRecentSearch` y `deleteRecentSearch`; `websocket_actions.ts` maneja `recent_search_updated`. |
| Servidor | Contenedor | Go | `api4/recent_search.go` (rutas y `requireRecentSearchOwner`), `app/recent_search.go` (reglas de negocio y publicación del evento), `store/sqlstore/recent_search_store.go` (upsert con `ON CONFLICT`, orden por `updateat`, poda a cinco). |
| PostgreSQL | Contenedor (base de datos) | PostgreSQL 15 | Tabla `recentsearchqueries` creada por la migración 000233 y ajustada por la 000234 (columnas `VARCHAR(26)` e índice único sobre `md5(terms)`). |

## Relaciones

| Origen | Destino | Protocolo | Descripción |
|---|---|---|---|
| Miembro del equipo | Webapp | HTTPS | Usa el panel de búsqueda. |
| Webapp | Servidor | REST, JSON | `GET`, `POST` y `DELETE` en `/api/v4/users/{user_id}/recent_searches`. |
| Servidor | Webapp | WebSocket | `recent_search_updated` con `action` `save` (item) o `delete` (id), enviado solo al usuario dueño. |
| Servidor | PostgreSQL | SQL | Lee y escribe el historial del usuario autenticado. |

## Alcance de esta vista

Se limita a los contenedores que participan en las búsquedas recientes. La aplicación móvil queda fuera del alcance. El proceso de validación (Team CI en GitHub Actions) se describe en el documento del segundo entregable y en `.github/workflows/team-ci.yml`.

[Volver al README](../../README.md#diagrama-de-arquitectura-c4)

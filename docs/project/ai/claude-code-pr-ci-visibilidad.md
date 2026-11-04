# Claude Code · PR de CI · Que las pruebas del store siempre corran y se vean por nombre

**Repositorio:** `danielcanessa/mattermost`
**Responsable:** Roberto (`rdelcidw`), autor del workflow de CI (PR #3)
**Rama:** `ci/recent-search-test-visibility`, desde `master` → PR hacia `master`

Esta vez **sí** se modifica `.github/workflows/team-ci.yml`, pero solo el job `server`. El resto de las reglas del PR 2 siguen igual:
- Todo va a `--repo danielcanessa/mattermost --base master`.
- Ok antes de push, issue o PR.
- Nada de force push ni de mencionar el repo oficial.
- Commits con `-s` y la identidad de `rdelcidw`.

## Los dos problemas
1. **Las pruebas del store no corren si solo cambia `storetest`.** Están escritas en `channels/store/storetest`, pero solo se ejecutan desde el wrapper `TestRecentSearchStore` de `channels/store/sqlstore`. Si un PR cambia solo `storetest` (es el caso del PR de pruebas de orden de Daniel), el CI corre `go test ./channels/store/storetest`, que no ejecuta ninguna prueba del store, y el check sale en verde igual.
2. **El log no muestra los nombres de las pruebas.** Como `go test` corre sin `-v`, cuando todo pasa el log solo muestra `ok <paquete>` y no se puede confirmar qué pruebas corrieron. Lo viste en la corrida #10.

## Cambios en el job `server` de `team-ci.yml`
1. **Paso `Changed packages`:** si la lista incluye `./channels/store/storetest`, que agregue también `./channels/store/sqlstore` (sin duplicar; un `sort -u` al final sirve). La suite completa de sqlstore tarda unos 90 s, así que no hace falta filtrarla.
2. **Paso `go test`:** que en el log aparezca cada prueba que contiene `RecentSearch` con su resultado (`--- PASS` o `--- FAIL`), en todos los paquetes, incluidas las subpruebas como `TestRecentSearchStore/PostgreSQL/Limits`. Condiciones:
   - **El paso tiene que seguir fallando cuando falla una prueba.** Si usás tuberías, ojo con perder el código de salida: el shell por defecto de GitHub usa `-eo pipefail`, pero verificalo.
   - **Si una prueba falla, el detalle del error completo tiene que seguir en el log.**
   - **El log de la suite completa de sqlstore no puede volverse ilegible.** Con `-v` son miles de líneas. Proponé el enfoque antes de implementarlo; por ejemplo, `-json` más un resumen filtrado, o `-v` guardado en un archivo y subido como artefacto, imprimiendo solo un resumen.
   - **No agregués dependencias externas sin consultarme.** `jq` ya viene en los runners de GitHub, así que se puede usar.
3. **No cambiés nada más:** ni los triggers, ni el job `webapp`, ni los servicios, ni las versiones fijadas de las acciones, ni el filtro `-run RecentSearch` de api4 y app.

## Verificación
- Antes de proponer el cambio, probá la lógica del paso `Changed packages` en tu máquina con listas de archivos de ejemplo: solo storetest, solo sqlstore, los dos, ninguno.
- Probá el script del paso `go test` localmente o en el contenedor (`golang:1.26.7`) con dos casos:
  - Una prueba que pasa.
  - Una prueba que falla a propósito. Sin commitear: el paso tiene que salir con código distinto de 0 y mostrar el error.
- En el PR, el cambio de `team-ci.yml` dispara el job `server` aunque no cambie código Go; revisá cómo lo maneja el job `changes`. Si con ese PR el job `server` no tiene paquetes que probar, decime cómo demostrar que funciona. Una opción es comparar la corrida del PR 5 antes y después de traer este cambio a esa rama con merge, pero no lo hagás sin consultarme.

## Issue y PR (con mi ok)
- Issue: "CI: always run store tests when storetest changes and list RecentSearch test names". Explicá los dos problemas y poné como evidencia la corrida #10 (https://github.com/danielcanessa/mattermost/actions/runs/40945553718).
- Commit: `ci: run store tests on storetest changes and list recent search test names`.
- Cuerpo del PR según la plantilla, con `Closes #<N>`, `Release Note` `NONE` y al final «Cualquier integrante del equipo puede revisarlo y aprobarlo». Sin revisor asignado.

## Reporte final
URL del issue, URL del PR, URL de la corrida y un fragmento del log con los nombres de las pruebas `RecentSearch`.

# Uso de herramientas de IA en el proyecto

Esta carpeta conserva los archivos de instrucciones que el equipo Foo usó con Claude Code para ejecutar los pull requests del proyecto, como evidencia del uso de IA solicitado en la consigna del curso.

Regla de trabajo en todos los casos: la IA propone y ejecuta en la máquina de un integrante; esa persona revisa el diff y aprueba cada acción con efecto fuera de su máquina (push, issue, pull request); otro integrante revisa y aprueba el PR.

| Archivo | PR | Contenido principal |
|---|---|---|
| claude-code-pr1-ci-workflow.md | #3 | Workflow Team CI: jobs changes, webapp y server; acciones fijadas por SHA; reglas de fork y de aprobación previa. |
| claude-code-pr2-persistencia-roberto.md | #5 | Reset de id y fechas en la capa app; migración 000234; pruebas de límites y aislamiento; verificación up → down → up. |
| claude-code-pr-ci-visibilidad.md | #7 | Ejecutar sqlstore cuando cambia storetest; nombres de pruebas en el log con `go test -json`. |
| claude-code-pr3-orden-daniel.md | #9 | Pruebas de orden en store y webapp; evidencia de regresión; textos de error; `make store-layers`. |
| claude-code-pr4-integracion-braulio.md | #10 | Contrato de la API; prueba de eventos WebSocket con dos usuarios; pruebas del manejador en la webapp. |
| claude-code-ajustes-pr10.md | #10 | Creación del issue #11 y corrección del título y la descripción del PR #10. |
| instrucciones-roger-pr1.txt | #1 | Instrucciones para una persona: cierre del PR externo, traer master con merge, limpieza de la rama. |

Otras herramientas usadas: Cursor con la plantilla lidr-specboot y OpenSpec (análisis y diseño en `openspec/changes/recent-searches/`) y la revisión automática de código CodeRabbit sobre el PR de la implementación.

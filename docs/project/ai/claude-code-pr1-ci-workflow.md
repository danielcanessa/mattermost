# Claude Code · PR 1 · Workflow de CI del equipo

**Repositorio:** `danielcanessa/mattermost`, fork público de `mattermost/mattermost`
**Responsable:** Roberto (`rdelcidw`)
**Rama:** `feature/ci-workflow` → `master` del fork
**Base:** documento final del entregable 2, sección 2.4: un workflow que corre en `pull_request` y en push a la rama base, con formato/lint, chequeo de tipos/compilación y pruebas del módulo afectado.

Idioma: comentarios en el PR y en el issue en español; nombres de archivos, jobs, commits y código en inglés.

---

## Reglas (no negociables)

1. **El PR va al fork del equipo, nunca a `mattermost/mattermost`.** `gh pr create` en un fork propone el repo original por defecto, y ese error ya pasó una vez (PR #39343). Siempre usá `--repo danielcanessa/mattermost --base master`, y antes de crear el PR mostrame el comando completo.
2. No hagás `push`, no creés issues ni PR, y no cambiés configuración de GitHub sin mostrarme antes qué vas a hacer y recibir mi "ok".
3. Nunca `push --force` y nunca commits directos a `master`.
4. No modifiqués ni borrés workflows de upstream. Los que hay que apagar se deshabilitan desde GitHub (Fase B), no desde el código.
5. Usá los scripts, acciones y versiones que ya tiene Mattermost; no inventés comandos. Si algo de este documento no coincide con lo que encontrás en el repo, parás y me lo reportás.
6. Commits en inglés, en Conventional Commits y firmados con `-s`, como hace el equipo.

---

## Estado actual del fork (26/10/2026)

- **Actions acaba de habilitarse.** Con eso quedaron activos los más de 50 workflows de Mattermost (API, Server CI, CodeQL, Claude Code, Docs CI, etc.). Todavía no hay ninguna corrida.
- **Issues está deshabilitado** (`has_issues: false`). Solo Daniel, como dueño, puede activarlo en Settings → General → Features → Issues.
- Daniel me agregó como colaborador.
- Hay un PR abierto de Roger (#1, `feature/recent-searches` → `master`). Roger no va a hacer push hasta que terminemos la Fase B.

**Prioridad:** apagar los workflows de Mattermost (Fase B) antes de cualquier push, issue o PR. Cualquier evento en el repo los dispararía, fallarían por secretos e infraestructura que el fork no tiene, y dejarían corridas en rojo en el historial de Actions.

---

## Fase A · Verificación previa (solo lectura)

```bash
git remote -v
git fetch origin
git log -1 --format='%h %s' origin/master
gh auth status
gh api repos/danielcanessa/mattermost --jq '{fork, parent: .parent.full_name, has_issues, permissions}'
gh api repos/danielcanessa/mattermost/actions/permissions
gh workflow list --repo danielcanessa/mattermost --all --limit 200 --json path,state --jq 'group_by(.state) | map({state: .[0].state, count: length})'
gh run list --repo danielcanessa/mattermost --limit 20
```

Reportame:

- Si `origin` apunta a `danielcanessa/mattermost`. Si apunta a `mattermost/mattermost`, parás.
- Mis permisos. Necesito `push: true`; si sale `false`, probablemente no acepté la invitación de colaborador todavía (https://github.com/danielcanessa/mattermost/invitations). Parás y me avisás.
- Que Actions diga `"enabled": true`.
- Cuántos workflows hay activos y deshabilitados.
- Si ya hay corridas. Si alguna está en curso, la cancelamos en la Fase B.

De upstream, anotá estos valores para usarlos tal cual:

- SHAs fijados de `actions/checkout`, `actions/setup-go` y `actions/upload-artifact`:
  `grep -rhoE 'uses: actions/[a-z-]+@[a-f0-9]{40} # v[0-9.]+' .github/workflows | sort | uniq -c | sort -rn`
  Referencia: `actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2`.
- La imagen de PostgreSQL que usa upstream para pruebas, en `server/build/docker-compose*.yml`.
- Si las pruebas de Go necesitan `go.work`: revisá el target `setup-go-work` en `server/Makefile`.
- Los scripts de la webapp en `webapp/channels/package.json`: `check`, `check-types` y `test-ci`.

---

## Fase B · Apagar los workflows de Mattermost y crear el issue (mostrame antes de ejecutar)

1. **Deshabilitar todos los workflows de upstream.** Esto evita que corran en cada PR y en cada push a `master` (riesgo R8 del documento: consumo de minutos). Los archivos no se tocan: solo se apagan desde GitHub. El script es idempotente:

   ```bash
   gh workflow list --repo danielcanessa/mattermost --all --limit 200 --json path,state \
     --jq '.[] | select(.path != ".github/workflows/team-ci.yml") | select(.state == "active") | .path' |
   while read -r wf; do
     echo "disable: $wf"
     gh workflow disable "$wf" --repo danielcanessa/mattermost
   done
   ```

   Verificación: `gh workflow list --repo danielcanessa/mattermost` (sin `--all`) debe salir vacío. Si algún `disable` falla con 403 o 404, parás y me mostrás el error; no sigás con otra estrategia.

2. **Cancelar corridas en curso**, si la Fase A mostró alguna:

   ```bash
   gh run list --repo danielcanessa/mattermost --status in_progress --json databaseId --jq '.[].databaseId' |
   xargs -r -n1 gh run cancel --repo danielcanessa/mattermost
   gh run list --repo danielcanessa/mattermost --status queued --json databaseId --jq '.[].databaseId' |
   xargs -r -n1 gh run cancel --repo danielcanessa/mattermost
   ```

   Cuando esto termine, avisame que Roger ya puede hacer push.

3. **Crear el issue de esta tarea**, como pide la sección 2.2 del documento:

   ```bash
   gh issue create --repo danielcanessa/mattermost \
     --title "CI: configure GitHub Actions workflow for pull requests" \
     --body "Configurar el workflow de CI descrito en la sección 2.4 del entregable 2: formato/lint, chequeo de tipos/compilación y pruebas del módulo afectado en cada PR hacia master y feature/recent-searches.

   Criterios de aceptación:
   - [ ] Corre en pull_request hacia master y feature/recent-searches, y en push a master.
   - [ ] Solo ejecuta el job del módulo que cambió (webapp o server).
   - [ ] Permisos mínimos y acciones fijadas por SHA.
   - [ ] El PR del workflow muestra una corrida exitosa."
   ```

   Guardá el número del issue (`<N>`).

---

## Fase C · El workflow

```bash
git switch -c feature/ci-workflow origin/master
```

Creá **un solo archivo**: `.github/workflows/team-ci.yml`. El prefijo `team-` evita chocar con los workflows de Mattermost.

Diseño:

- **Disparadores:** `pull_request` hacia `master` y `feature/recent-searches`, y `push` a `master`.
- **Job `changes`:** detecta si el cambio toca `webapp/`, `server/` o el propio workflow, con `git diff` contra la base. Los demás jobs usan `if:` según ese resultado. No usés `paths:` a nivel de workflow: un check obligatorio que no corre queda pendiente para siempre, mientras que un job omitido con `if:` cuenta como exitoso.
- **Job `webapp`:** usa la acción local `./.github/actions/webapp-setup` de upstream. Corre `npm run check`, `npm run check-types` y Jest sobre `src/components/new_search` más las pruebas relacionadas con los archivos cambiados (`--changedSince`). Sube la cobertura como artefacto.
- **Job `server`:** PostgreSQL como contenedor de servicio con las mismas credenciales que usa upstream en sus pruebas. Corre `gofmt`, `go vet` y `go test` **solo de los paquetes modificados**.
- **Permisos:** `contents: read` y nada más.
- **Concurrencia:** cancela corridas viejas del mismo PR.

Esqueleto. Reemplazá los `<...>` con lo que anotaste en la Fase A:

```yaml
name: Team CI
on:
  pull_request:
    branches: [master, feature/recent-searches]
  push:
    branches: [master]

concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

permissions:
  contents: read

jobs:
  changes:
    runs-on: ubuntu-24.04
    outputs:
      webapp: ${{ steps.diff.outputs.webapp }}
      server: ${{ steps.diff.outputs.server }}
    steps:
      - uses: actions/checkout@<SHA-checkout>
        with:
          persist-credentials: false
          fetch-depth: 0
      - id: diff
        env:
          BASE: ${{ github.event_name == 'pull_request' && format('origin/{0}', github.base_ref) || github.event.before }}
        run: |
          files=$(git diff --name-only "$BASE"...HEAD)
          echo "$files"
          wf=$(echo "$files" | grep -c '^.github/workflows/team-ci.yml$' || true)
          echo "webapp=$([ "$wf" -gt 0 ] || echo "$files" | grep -q '^webapp/' && echo true || echo false)" >> "$GITHUB_OUTPUT"
          echo "server=$([ "$wf" -gt 0 ] || echo "$files" | grep -q '^server/' && echo true || echo false)" >> "$GITHUB_OUTPUT"

  webapp:
    needs: changes
    if: needs.changes.outputs.webapp == 'true'
    runs-on: ubuntu-24.04
    timeout-minutes: 45
    defaults:
      run:
        working-directory: webapp/channels
    steps:
      - uses: actions/checkout@<SHA-checkout>
        with:
          persist-credentials: false
          fetch-depth: 0
      - uses: ./.github/actions/webapp-setup
      - name: Lint
        run: npm run check
      - name: Types
        run: npm run check-types
      - name: Tests (new_search + related)
        env:
          BASE_REF: ${{ github.base_ref || 'master' }}
        run: >
          npx cross-env TZ=Etc/UTC LC_ALL=en_US.UTF-8 LANG=en_US.UTF-8
          jest --ci --coverage --coverageReporters=text-summary --coverageReporters=json-summary
          --findRelatedTests $(git diff --name-only "origin/${BASE_REF}"...HEAD -- . | sed 's#^webapp/channels/##' | grep -E '\.(ts|tsx)$' || true)
          src/components/new_search
      - uses: actions/upload-artifact@<SHA-upload-artifact>
        if: always()
        with:
          name: webapp-coverage
          path: webapp/channels/coverage
          retention-days: 14

  server:
    needs: changes
    if: needs.changes.outputs.server == 'true'
    runs-on: ubuntu-24.04
    timeout-minutes: 45
    services:
      postgres:
        image: <imagen de PostgreSQL de upstream>
        env:
          POSTGRES_USER: mmuser
          POSTGRES_PASSWORD: mostest_password
          POSTGRES_DB: mattermost_test
        ports: ["5432:5432"]
        options: >-
          --health-cmd "pg_isready -U mmuser"
          --health-interval 10s --health-timeout 5s --health-retries 10
    env:
      MM_SQLSETTINGS_DRIVERNAME: postgres
      MM_SQLSETTINGS_DATASOURCE: postgres://mmuser:mostest_password@localhost:5432/mattermost_test?sslmode=disable&connect_timeout=10
      TEST_DATABASE_POSTGRESQL_DSN: postgres://mmuser:mostest_password@localhost:5432/mattermost_test?sslmode=disable&connect_timeout=10
    defaults:
      run:
        working-directory: server
    steps:
      - uses: actions/checkout@<SHA-checkout>
        with:
          persist-credentials: false
          fetch-depth: 0
      - uses: actions/setup-go@<SHA-setup-go>
        with:
          go-version-file: server/go.mod
      # Si server/Makefile lo requiere: - run: make setup-go-work
      - name: Changed packages
        id: pkgs
        env:
          BASE: ${{ github.event_name == 'pull_request' && format('origin/{0}', github.base_ref) || github.event.before }}
        run: |
          files=$(git diff --name-only "$BASE"...HEAD -- . | grep '\.go$' | sed 's#^server/##' || true)
          echo "files=$(echo $files)" >> "$GITHUB_OUTPUT"
          echo "list=$(echo "$files" | xargs -r -n1 dirname | sort -u | sed 's#^#./#' | tr '\n' ' ')" >> "$GITHUB_OUTPUT"
      - name: gofmt
        if: steps.pkgs.outputs.list != ''
        run: test -z "$(gofmt -l ${{ steps.pkgs.outputs.files }})" || { gofmt -l ${{ steps.pkgs.outputs.files }}; exit 1; }
      - name: go vet
        if: steps.pkgs.outputs.list != ''
        run: go vet ${{ steps.pkgs.outputs.list }}
      - name: go test
        if: steps.pkgs.outputs.list != ''
        run: go test -count=1 -coverprofile=coverage.out ${{ steps.pkgs.outputs.list }}
      - uses: actions/upload-artifact@<SHA-upload-artifact>
        if: always() && steps.pkgs.outputs.list != ''
        with:
          name: server-coverage
          path: server/coverage.out
          retention-days: 14
```

Notas para vos:

- El esqueleto es un punto de partida. Revisá que la lógica de `changes` y de los paquetes modificados funcione con nombres de archivo y rutas reales, y si la corregís, contame qué cambiaste.
- Este PR solo agrega el workflow. Como cambia `.github/workflows/team-ci.yml`, los jobs `webapp` y `server` corren igual, y eso sirve de prueba del pipeline completo. Si el job `server` no encuentra paquetes Go modificados, sus pasos se omiten: está bien.
- Si alguna prueba de Mattermost necesita servicios extra (MinIO, LDAP, etc.) para correr, no lo parchés: reportámelo.

---

## Fase D · Verificación antes del commit

Mostrame la salida de:

```bash
yamllint -c .yamllint .github/workflows/team-ci.yml
actionlint .github/workflows/team-ci.yml     # si no está instalado: go install github.com/rhysd/actionlint/cmd/actionlint@latest
git diff --name-status origin/master...HEAD
```

El `git diff` solo debe listar `.github/workflows/team-ci.yml`.

---

## Fase E · Commit, push y PR (con mi ok)

```bash
git add .github/workflows/team-ci.yml
git commit -s -m "ci: add team CI workflow for pull requests"
git push -u origin feature/ci-workflow
```

Mostrame este comando completo antes de ejecutarlo:

```bash
gh pr create --repo danielcanessa/mattermost --base master --head feature/ci-workflow \
  --title "ci: add team CI workflow for pull requests" \
  --body-file <archivo temporal>
```

Cuerpo del PR, siguiendo `.github/PULL_REQUEST_TEMPLATE.md`:

- **Summary:** agrega el workflow de CI del equipo (sección 2.4 del entregable 2): lint, tipos y pruebas del módulo afectado en PR hacia `master` y `feature/recent-searches`.
- **Ticket Link:** `Closes #<N>`.
- **Screenshots:** no aplica.
- **Release Note:** `NONE`.
- Una línea al final: «Cualquier integrante del equipo puede revisarlo y aprobarlo».

Después de crear el PR:

1. Confirmá con `gh pr view --repo danielcanessa/mattermost --json url,baseRefName,headRefName` que la URL es de `danielcanessa/mattermost` y la base es `master`.
2. Seguí la corrida con `gh run watch --repo danielcanessa/mattermost` y reportame el resultado de cada job.
3. No asignés un revisor específico: cualquier integrante del equipo lo puede revisar y aprobar, salvo yo, que soy el autor. Mostrame la URL del PR para pasarla al grupo.

---

## Reporte final que necesito

| Elemento | Valor |
| --- | --- |
| Issue | #N (URL) |
| PR | #M (URL), base `master` del fork |
| Corrida de Actions | URL y resultado de `changes`, `webapp` y `server` |
| Workflows de upstream deshabilitados | Cantidad |
| Desviaciones respecto de este documento | Lista |

Recordatorio para el equipo: cuando este PR esté integrado, Roger hace el ajuste de su rama (traer `master` y limpiar archivos), y recién ahí Roberto, Daniel y Braulio crean sus ramas de pruebas.

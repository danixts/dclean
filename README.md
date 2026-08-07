# dclean

TUI para limpiar directorios temporales, cache de desarrollo y basura del sistema en Linux.

Escanea multiples rutas configurables, caches de herramientas, Docker y basura del sistema
(revisiones viejas de snap, cache de APT, journal, temporales rancios, crash dumps), agrupa
los resultados por categoria, muestra el espacio ocupado y permite eliminar de forma
selectiva o masiva. Persiste rutas e historial en SQLite.

Nunca toca caches ni perfiles de navegador.

```
> [x] ▾ Go Build Cache     10.3 GB  (4 dirs)
        /home/user/.cache/go-build   9.9 GB
        /home/user/.cache/goimports  283.2 MB
  [ ] ▸ Snap Cache           4.3 GB  (6 dirs)
  [ ] ▸ Snap Old Revisions   3.9 GB  (2 dirs)
  [ ] ▸ IDE Cache            2.5 GB  (1 dirs)
  [ ] ▸ Package Manager      1.5 GB  (3 dirs)

  Selected: 10.3 GB (4 dirs)

  space: toggle  a: all  tab: expand  enter/d: delete  p: paths  q: quit
```

## Requisitos

- Go 1.26+
- Linux (soporte para snap y `.cache`)
- gcc (requerido por `go-sqlite3`)

## Instalacion

```bash
git clone https://github.com/danyjs/dclean.git
cd dclean
make install
```

El binario se instala en `$(go env GOPATH)/bin/dclean`.

## Uso

### TUI interactiva

```bash
dclean
```

#### Pantalla principal — Seleccion de grupos

Muestra todos los directorios encontrados agrupados por categoria, ordenados por tamano.

| Tecla | Accion |
|-------|--------|
| `j` / `k` / flechas | Navegar entre grupos |
| `space` | Seleccionar/deseleccionar grupo |
| `a` | Seleccionar/deseleccionar todo |
| `tab` / `e` | Expandir grupo para ver directorios individuales |
| `enter` / `d` | Ir a pantalla de confirmacion |
| `y` | Confirmar eliminacion |
| `n` / `esc` | Cancelar eliminacion |
| `p` | Abrir gestion de rutas |
| `q` | Salir |

#### Pantalla de rutas — CRUD interactivo

Accesible desde la pantalla principal con `p`. Muestra todas las rutas configuradas con su estado.

```
dclean — Scan Paths

> [active]   /home/user/Documentos  (Documentos)
  [inactive] /home/user/Projects    (Projects)
  [active]   /home/user/dev         (dev)

  space: toggle  a: add  x: delete  esc: back
```

| Tecla | Accion |
|-------|--------|
| `j` / `k` / flechas | Navegar entre rutas |
| `space` | Activar/desactivar ruta |
| `a` | Agregar nueva ruta (abre input de texto) |
| `x` | Eliminar ruta seleccionada |
| `esc` / `q` | Volver a pantalla principal (re-escanea si hubo cambios) |

Al agregar una ruta se valida que el directorio exista. Al volver a la pantalla principal se re-escanean solo las rutas activas.

### Modo no-interactivo

```bash
# Listar todo lo que se puede limpiar
dclean --list

# Ver sin eliminar (dry-run)
dclean --dry-run
```

### Gestion de rutas por CLI

```bash
# Ver rutas configuradas
dclean --paths

# Agregar una ruta
dclean --add /home/user/workspace

# Eliminar una ruta
dclean --remove /home/user/workspace
```

En la primera ejecucion se detecta automaticamente el usuario del sistema y se agregan las rutas que existan: `~/Documentos`, `~/Documents`, `~/Projects`, `~/dev`.

### Historial de eliminaciones

Cada directorio eliminado se registra en SQLite con su categoria, tamano y fecha.

```bash
dclean --history
```

```
Deletion history by category:
  Go Build Cache              10.3 GB  (4 dirs, last: 2026-03-17 22:30:00)
  Snap Cache                   4.3 GB  (6 dirs, last: 2026-03-17 22:30:00)
  Snap Old Revisions           3.9 GB  (2 dirs, last: 2026-03-17 22:30:00)

  Total freed: 18.5 GB
```

## Categorias detectadas

### Directorios de proyecto (escaneo recursivo)

| Categoria | Directorios |
|-----------|-------------|
| Node Modules | `node_modules` |
| Package Manager Store | `.npm`, `.pnpm-store`, `.yarn`, `.bun` |
| Next.js | `.next` |
| Turborepo | `.turbo` |
| Build Output | `dist`, `build`, `.output`, `.nuxt`, `.svelte-kit`, `.angular` |
| Dev Cache | `.parcel-cache`, `.vite`, `.eslintcache`, `.temp` |
| Test Coverage | `coverage`, `.nyc_output` |
| Python Cache | `__pycache__`, `.pytest_cache`, `.mypy_cache`, `.ruff_cache` |

### Cache del sistema (`~/.cache`)

| Categoria | Directorios |
|-----------|-------------|
| Go Build Cache | `go-build`, `gopls`, `goimports`, `golangci-lint` |
| IDE Cache | `JetBrains`, `cursor-compile-cache` |
| Package Manager Cache | `pip`, `pnpm`, `yarn`, `uv`, `npm`, `turbo`, `pre-commit`, `virtualenv` |
| Dev Tools Cache | `typescript`, `eslint`, `prettier`, `ms-playwright`, `helm`, `opencode` |
| AI Tools Cache | `codebase-memory-mcp`, `claude-cli-nodejs`, `kimi-code` |
| System Cache | `thumbnails`, `tracker3`, `fontconfig`, `nvidia`, `mesa_shader_cache` |

Los caches de navegador quedan explicitamente fuera del scan: guardan datos de sesion y borrarlos cierra tus sesiones.

### Caches por herramienta (fuera de `~/.cache`)

| Categoria | Ruta |
|-----------|------|
| Go Module Cache | `~/go/pkg/mod` |
| Maven Repository | `~/.m2/repository` |
| npm Cache | `~/.npm/_cacache` |
| Bun Cache | `~/.bun/install/cache` |
| Cargo Registry | `~/.cargo/registry` |
| pnpm Store | `~/.local/share/pnpm/store` |
| Gradle Cache | `~/.gradle/caches`, `~/.gradle/daemon` |
| Pyppeteer Chromium | `~/.local/share/pyppeteer/local-chromium` |
| Trash | `~/.local/share/Trash` |

### Snap (`~/snap`)

| Categoria | Deteccion |
|-----------|-----------|
| Snap Old Revisions | Revisiones numeradas que no son la activa (symlink `current`) |
| Snap Cache | `~/snap/<app>/common/.cache` mayores a 1 MB (navegadores excluidos) |

### Basura del sistema

Requiere privilegios: se ejecuta via `sudo -n`, sin prompt interactivo. Si no hay sudo sin
contrasena disponible, los items aparecen en el scan pero fallan al eliminarse. Ver
[Permisos de sistema](#permisos-de-sistema).

| Categoria | Deteccion | Accion |
|-----------|-----------|--------|
| Snap Disabled Revisions | Revisiones viejas en `/var/lib/snapd/snaps` que snapd conserva tras cada update | `snap remove --revision` |
| APT Package Cache | `.deb` descargados en `/var/cache/apt/archives` | `apt-get clean` |
| APT Orphaned Packages | Paquetes huerfanos, kernels viejos incluidos | `apt-get -y autoremove --purge` |
| System Journal | Logs de systemd por encima de 100 MB | `journalctl --vacuum-size=100M` |
| Stale Temp Files | Entradas de `/tmp` y `/var/tmp` propias sin tocar hace 7+ dias | borrado directo |
| Crash Reports | `/var/crash` y `/var/lib/systemd/coredump` | borrado directo |

Se ignoran los sockets y directorios de sesion (`.X11-unix`, `systemd-private-*`,
`snap-private-tmp`, etc), y solo se toca lo que pertenece a tu usuario.

## Permisos de sistema

Para que dclean limpie la basura del sistema sin pedir contrasena, autorizá solo esos
comandos en sudoers:

```bash
sudo tee /etc/sudoers.d/dclean > /dev/null <<'EOF'
%sudo ALL=(root) NOPASSWD: /usr/bin/apt-get clean, /usr/bin/apt-get -y autoremove --purge, /usr/bin/journalctl --vacuum-size=100M, /usr/bin/snap remove --revision *
EOF
sudo chmod 0440 /etc/sudoers.d/dclean
sudo visudo -c
```

Alternativa sin sudoers: `sudo dclean`.

## Estructura del proyecto

```
dclean/
├── cmd/
│   └── main.go                  # Entrypoint, CLI flags, comandos
├── internal/
│   ├── domain/
│   │   ├── types.go             # Tipos: FoundDir, ScanPath, Category, DeletionRecord
│   │   ├── categories.go        # Definicion de categorias y targets
│   │   └── format.go            # FormatSize (bytes a human-readable)
│   ├── scanner/
│   │   ├── scanner.go           # MultiScanner: recursivo, directo, snap
│   │   ├── docker.go            # Volumenes huerfanos y system prune
│   │   └── system.go            # Basura del sistema: snap, apt, journal, tmp, crashes
│   ├── store/
│   │   └── store.go             # SQLite: rutas, historial, migraciones
│   └── tui/
│       ├── model.go             # Bubble Tea model, update, key handlers, grupos
│       ├── view.go              # Renderizado: select, paths, confirm, done
│       └── styles.go            # Estilos lipgloss
├── .gitignore
├── Makefile
├── go.mod
└── README.md
```

## Makefile

```bash
make build      # Compilar binario
make install    # Compilar e instalar en GOPATH/bin
make run        # Compilar y ejecutar
make clean      # Eliminar binario
make deps       # go mod tidy
make update     # Actualizar todas las dependencias
```

## Persistencia

SQLite en `~/.config/dclean/dclean.db` con dos tablas:

**scan_paths**

| Columna | Tipo | Descripcion |
|---------|------|-------------|
| id | INTEGER | PK autoincrement |
| path | TEXT | Ruta absoluta (unique) |
| label | TEXT | Nombre corto para mostrar |
| active | INTEGER | 1 = activa, 0 = inactiva |
| created_at | TEXT | Fecha de creacion |

**deletion_history**

| Columna | Tipo | Descripcion |
|---------|------|-------------|
| id | INTEGER | PK autoincrement |
| path | TEXT | Ruta del directorio eliminado |
| category | TEXT | Categoria (Node Modules, Turborepo, etc) |
| size_bytes | INTEGER | Tamano en bytes al momento de eliminar |
| deleted_at | TEXT | Fecha de eliminacion |

## Dependencias

- [bubbletea](https://github.com/charmbracelet/bubbletea) — framework TUI
- [bubbles](https://github.com/charmbracelet/bubbles) — componentes TUI (textinput)
- [lipgloss](https://github.com/charmbracelet/lipgloss) — estilos de terminal
- [go-sqlite3](https://github.com/mattn/go-sqlite3) — driver SQLite

## Licencia

MIT

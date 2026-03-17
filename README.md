# dclean

TUI para limpiar directorios temporales, cache de desarrollo y revisiones viejas de snap en Linux.

Escanea multiples rutas configurables, agrupa los resultados por categoria, muestra el espacio ocupado y permite eliminar de forma selectiva o masiva.

```
> [x] ▾ Go Build Cache     10.3 GB  (4 dirs)
        /home/user/.cache/go-build   9.9 GB
        /home/user/.cache/goimports  283.2 MB
  [ ] ▸ Snap Cache           4.3 GB  (6 dirs)
  [ ] ▸ Snap Old Revisions   3.9 GB  (2 dirs)
  [ ] ▸ IDE Cache            2.5 GB  (1 dirs)
  [ ] ▸ Package Manager      1.5 GB  (3 dirs)

  Selected: 10.3 GB (4 dirs)

  space: toggle  a: all  tab: expand  enter/d: delete  q: quit
```

## Requisitos

- Go 1.26+
- Linux (soporte para snap y `.cache`)
- gcc (requerido por `go-sqlite3`)

## Instalacion

```bash
git clone https://dclean.git
cd dclean
make install
```

El binario se instala en `$(go env GOPATH)/bin/dclean`.

## Uso

### TUI interactiva

```bash
dclean
```

| Tecla               | Accion                                |
| ------------------- | ------------------------------------- |
| `j` / `k` / flechas | Navegar entre grupos                  |
| `space`             | Seleccionar/deseleccionar grupo       |
| `a`                 | Seleccionar/deseleccionar todo        |
| `tab` / `e`         | Expandir grupo para ver directorios   |
| `enter` / `d`       | Confirmar eliminacion                 |
| `y`                 | Confirmar en pantalla de confirmacion |
| `n` / `esc`         | Cancelar                              |
| `q`                 | Salir                                 |

### Modo no-interactivo

```bash
# Listar todo lo que se puede limpiar
dclean --list

# Ver sin eliminar (dry-run)
dclean --dry-run
```

### Gestion de rutas

Las rutas de escaneo se persisten en SQLite (`~/.config/dclean/dclean.db`).

En la primera ejecucion se detecta automaticamente el usuario del sistema y se agregan las rutas que existan: `~/Documentos`, `~/Documents`, `~/Projects`, `~/dev`.

```bash
# Ver rutas configuradas
dclean --paths

# Agregar una ruta
dclean --add /home/user/workspace

# Eliminar una ruta
dclean --remove /home/user/workspace
```

### Historial de eliminaciones

Cada eliminacion se registra en SQLite agrupada por categoria y directorio.

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

| Categoria             | Directorios                                                    |
| --------------------- | -------------------------------------------------------------- |
| Node Modules          | `node_modules`                                                 |
| Package Manager Store | `.npm`, `.pnpm-store`, `.yarn`, `.bun`                         |
| Next.js               | `.next`                                                        |
| Turborepo             | `.turbo`                                                       |
| Build Output          | `dist`, `build`, `.output`, `.nuxt`, `.svelte-kit`, `.angular` |
| Dev Cache             | `.parcel-cache`, `.vite`, `.eslintcache`, `.temp`              |
| Test Coverage         | `coverage`, `.nyc_output`                                      |
| Python Cache          | `__pycache__`, `.pytest_cache`, `.mypy_cache`, `.ruff_cache`   |

### Cache del sistema (`~/.cache`)

| Categoria             | Directorios                                                             |
| --------------------- | ----------------------------------------------------------------------- |
| Go Build Cache        | `go-build`, `gopls`, `goimports`, `golangci-lint`                       |
| IDE Cache             | `JetBrains`, `cursor-compile-cache`                                     |
| Package Manager Cache | `pip`, `pnpm`, `yarn`, `uv`, `npm`, `turbo`                             |
| Browser Cache         | `google-chrome`, `mozilla`, `BraveSoftware`, `microsoft-edge`           |
| Dev Tools Cache       | `typescript`, `eslint`, `prettier`, `ms-playwright`, `helm`, `opencode` |
| System Cache          | `thumbnails`, `tracker3`, `fontconfig`                                  |

### Snap (`~/snap`)

| Categoria          | Deteccion                                                     |
| ------------------ | ------------------------------------------------------------- |
| Snap Old Revisions | Revisiones numeradas que no son la activa (symlink `current`) |
| Snap Cache         | `~/snap/<app>/common/.cache` mayores a 1 MB                   |

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
│   │   └── scanner.go           # MultiScanner: recursivo, directo, snap
│   ├── store/
│   │   └── store.go             # SQLite: rutas, historial, migraciones
│   └── tui/
│       ├── model.go             # Bubble Tea model, update, grupos
│       ├── view.go              # Renderizado de cada pantalla
│       └── styles.go            # Estilos lipgloss
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
make update     # Actualizar dependencias
```

## Persistencia

SQLite en `~/.config/dclean/dclean.db` con dos tablas:

- **scan_paths**: rutas configuradas con label, estado activo/inactivo y fecha de creacion
- **deletion_history**: registro de cada directorio eliminado con path, categoria, tamanio y fecha

## Dependencias

- [bubbletea](https://github.com/charmbracelet/bubbletea) — framework TUI
- [lipgloss](https://github.com/charmbracelet/lipgloss) — estilos de terminal
- [go-sqlite3](https://github.com/mattn/go-sqlite3) — driver SQLite

## Licencia

MIT

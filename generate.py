#!/usr/bin/env python3
"""Render every Markdown file across the tour-planner monorepo to readable
HTML, grouped by project (and, within tour-agent-docs, by business case).
"""
import pathlib

import markdown

ROOT = pathlib.Path(__file__).parent
OUT_DIR = pathlib.Path("/output") if pathlib.Path("/output").is_dir() else ROOT / "docs-site-out"
EXTENSIONS = ["fenced_code", "tables", "toc", "nl2br"]

# Top-level project folders to pull docs from, in display order.
PROJECTS = [
    "tour-agent",
    "tour-planner-ai",
    "tour-planner-ui",
    "tour-commerce",
    "tour-planner-cms",
    "tour-agent-docs",
]

# Authored standalone HTML documents (not Markdown exports), copied verbatim
# and listed on the index. Keys are repo-relative paths.
STANDALONE_HTML = {
    "tour-agent-docs/architecture-dossier.html": "Architecture & Delivery Dossier",
}

EXCLUDE_DIR_NAMES = {
    ".git", ".venv", "venv", "node_modules", "dist", "build", ".next",
    "cdk.out", "coverage", "__pycache__", ".terraform", ".turbo",
}

STYLE = """:root { color-scheme: light dark; }
body {
  margin: 2rem auto;
  max-width: 900px;
  padding: 0 1rem 3rem;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
  line-height: 1.6;
}
a { text-decoration: none; }
a:hover { text-decoration: underline; }
pre, code { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace; }
pre { padding: 0.9rem; overflow: auto; border-radius: 8px; }
table { border-collapse: collapse; width: 100%; margin: 1rem 0; }
th, td { border: 1px solid #9996; padding: 0.5rem; text-align: left; }
blockquote { margin: 1rem 0; padding: 0.2rem 1rem; border-left: 4px solid #9996; }
hr { border: 0; border-top: 1px solid #9996; margin: 2rem 0; }
nav { margin-bottom: 2rem; }
.path { color: #888; font-size: 0.85em; }"""

PAGE_TEMPLATE = """<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{title}</title>
  <style>{style}</style>
</head>
<body>
{nav}{body}
</body>
</html>
"""

INDEX_TEMPLATE = """<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Tour Planner Docs</title>
  <style>{style}</style>
</head>
<body>
<h1>Tour Planner Docs</h1>
<p>HTML exports generated from Markdown files across every project in the monorepo.</p>
{sections}
</body>
</html>
"""


def humanize(name: str) -> str:
    return name.replace("-", " ").replace("_", " ").title()


def is_excluded(path: pathlib.Path) -> bool:
    return any(part in EXCLUDE_DIR_NAMES for part in path.parts)


def find_md_files(project_dir: pathlib.Path) -> list[pathlib.Path]:
    return sorted(p for p in project_dir.rglob("*.md") if not is_excluded(p.relative_to(project_dir)))


def render_page(md_path: pathlib.Path, depth: int) -> str:
    body = markdown.markdown(md_path.read_text(encoding="utf-8"), extensions=EXTENSIONS)
    home_href = "../" * depth + "index.html"
    nav = f'<nav><a href="{home_href}">Home</a></nav>'
    return PAGE_TEMPLATE.format(title=md_path.stem, style=STYLE, nav=nav, body=body)


def main() -> None:
    OUT_DIR.mkdir(parents=True, exist_ok=True)

    sections = []
    total = 0

    for project in PROJECTS:
        project_dir = ROOT / project
        if not project_dir.is_dir():
            continue
        md_files = find_md_files(project_dir)
        if not md_files:
            continue

        items = []
        for src_rel, label in STANDALONE_HTML.items():
            src = ROOT / src_rel
            if not src.is_file() or pathlib.Path(src_rel).parts[0] != project:
                continue
            out_rel = pathlib.Path(src_rel)
            (OUT_DIR / out_rel).parent.mkdir(parents=True, exist_ok=True)
            (OUT_DIR / out_rel).write_text(src.read_text(encoding="utf-8"), encoding="utf-8")
            items.append(f'<li><a href="{out_rel.as_posix()}"><b>{label}</b></a></li>')
            total += 1
        for md_path in md_files:
            rel = md_path.relative_to(project_dir)
            out_rel = pathlib.Path(project) / rel.with_suffix(".html")
            depth = len(out_rel.parts) - 1
            (OUT_DIR / out_rel).parent.mkdir(parents=True, exist_ok=True)
            (OUT_DIR / out_rel).write_text(render_page(md_path, depth), encoding="utf-8")
            subdir = rel.parent.as_posix()
            items.append(
                f'<li><a href="{out_rel.as_posix()}">{rel.stem}</a> '
                + (f'<span class="path">{subdir}</span>' if subdir != "." else "")
                + "</li>"
            )
            total += 1

        sections.append(f"<h2>{humanize(project)}</h2>\n<ul>\n" + "\n".join(items) + "\n</ul>")

    (OUT_DIR / "index.html").write_text(
        INDEX_TEMPLATE.format(style=STYLE, sections="\n".join(sections)), encoding="utf-8"
    )
    print(f"Rendered {total} markdown files from {len(PROJECTS)} projects to {OUT_DIR}")


if __name__ == "__main__":
    main()

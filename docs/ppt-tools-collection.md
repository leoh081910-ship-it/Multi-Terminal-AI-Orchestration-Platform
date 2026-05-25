# PPT 工具固定候选清单

本文仅基于任务指定的候选 `SKILL.md` 文件整理；未做额外技能发现或替代路径搜索。所有已包含工具均列出名称、路径、用途、最佳使用场景与入口命令或触发方式。

## 核心 PPT/PPTX 工具

### pptx

- **路径**：`/root/.codex/skills/pptx/SKILL.md`
- **用途**：处理任何与 `.pptx` 有关的任务，包括创建、读取、解析、编辑、拆分、合并演示文稿，以及处理模板、版式、讲者备注和评论。
- **最佳使用场景**：用户明确提供或要求输出 PowerPoint 文件，或提到 `deck`、`slides`、`presentation`、`.pptx` 文件名时，用作通用 PPTX 入口。
- **入口命令或触发**：触发词包括 `deck`、`slides`、`presentation`、`.pptx`；常用命令包括 `python -m markitdown presentation.pptx`、`python scripts/thumbnail.py presentation.pptx`、`python scripts/office/unpack.py presentation.pptx unpacked/`。

### ppt-master

- **路径**：`/workspace/projects/ppt-master-plus/skills/ppt-master/SKILL.md`
- **用途**：AI 驱动的多格式 SVG 内容生成系统，把 PDF、DOCX、URL、Markdown 等来源转为高质量 SVG 页面，并导出为 PPTX。
- **最佳使用场景**：用户要求“创建 PPT”“生成PPT”“做PPT”“制作演示文稿”，或需要从源文档经过策略、视觉设计、质量检查和后处理生成完整 PPTX。
- **入口命令或触发**：触发词包括 `create PPT`、`make presentation`、`生成PPT`、`做PPT`、`制作演示文稿`、`ppt-master`；关键脚本包括 `scripts/project_manager.py`、`scripts/source_to_md/ppt_to_md.py`、`scripts/finalize_svg.py`、`scripts/svg_to_pptx.py`。

### baoyu-slide-deck

- **路径**：`/root/.codex/skills/baoyu-slide-deck/SKILL.md`
- **用途**：从内容生成专业幻灯片图片，先产出大纲和风格说明，再逐页生成图片，并可合并为 PPTX 或 PDF。
- **最佳使用场景**：面向阅读和分享的图像型演示稿，例如社交媒体友好的长图式 deck、提案页、教学页或需要统一视觉风格的幻灯片图片。
- **入口命令或触发**：触发词包括 `create slides`、`make a presentation`、`generate deck`、`slide deck`、`PPT`；合并脚本包括 `scripts/merge-to-pptx.ts` 和 `scripts/merge-to-pdf.ts`，可配合 `--style`、`--audience`、`--lang`、`--slides` 等选项。

### visual-elements-ppt-rebuilder

- **路径**：`/root/.codex/skills/visual-elements-ppt-rebuilder/SKILL.md`
- **用途**：把扁平化的海报、画板、截图或导出的幻灯片拆成透明 PNG 视觉元素，并重建为标准可编辑 PPTX，文本以可编辑文本框复原。
- **最佳使用场景**：用户有一张已经合成的页面或截图，希望拆分素材、去背景、保留布局并生成可编辑 PowerPoint。
- **入口命令或触发**：触发场景包括拆分视觉元素、重建可编辑 PPT、把扁平海报或截图转为 PPTX；常用命令为 `python scripts/extract_elements.py --image source.png --spec layout-spec.json --out-dir ./output --prefix element --clean-prefix` 和 `python scripts/rebuild_editable_pptx.py --spec layout-spec.json --manifest ./output/elements_manifest.json --assets-dir ./output --out ./output/rebuilt_editable.pptx`。

### zone2ppt

- **路径**：`/root/.codex/skills/zone2ppt/SKILL.md`
- **用途**：把景观或规划图转为可编辑分区资产，包括底图、遮罩、功能分区 SVG 路径、拆分 SVG、组合 SVG 和可编辑 PPTX。
- **最佳使用场景**：景观功能分区图、zoning overlay、参考图边界提取、mask/base/zone 分层导出，以及修复规划分析图的空隙或重叠。
- **入口命令或触发**：触发词包括 `zone2ppt`、`景观功能分区图`、`zoning overlays`、`editable SVG/PPTX zoning blocks`；主命令为 `python "<skill>/scripts/generate_zoning.py" "plan.png" --reference "colored-zoning.png" --config "zones.json" --out "zoning_tool_output" --mode reference --analysis-roi "0,0.22,1,0.68"`，也支持 `--mode auto` 初稿。

## 辅助演示/设计工具

### guizang-ppt-skill

- **路径**：`/root/.codex/skills/guizang-ppt-skill/SKILL.md`
- **用途**：生成单文件横向网页 PPT，用于分享、演讲、发布会和 demo deck，支持电子杂志风与瑞士国际主义风。
- **最佳使用场景**：用户要网页 PPT、杂志风 PPT、瑞士风、horizontal swipe deck，且交付目标是可运行的单文件 HTML 而不是传统 PPTX。
- **入口命令或触发**：触发词包括 `网页 PPT`、`杂志风 PPT`、`瑞士风`、`Swiss Style`、`horizontal swipe deck`；入口流程是复制 `assets/template.html` 或 `assets/template-swiss.html` 到 `index.html`，生成本地 `images/`，瑞士风需运行 `node <SKILL_ROOT>/scripts/validate-swiss-deck.mjs path/to/index.html`。

### slidev

- **路径**：`/root/.codex/skills/slidev/SKILL.md`
- **用途**：使用 Slidev 框架创建和编辑基于 Markdown 的演示文稿，支持 Vue 组件、主题、布局、转场和逐步显示。
- **最佳使用场景**：开发者友好的 Markdown 演示稿、需要在 `packages/slides/` 中维护 `.slides.md` 或 `slides.md` 的项目，或需要交互式 Slidev 功能的 deck。
- **入口命令或触发**：触发场景包括创建新演示、编辑现有 slides、添加或修改内容、使用 Slidev 特性；启动命令为 `pnpm run slides [filename]`，默认服务地址为 `http://localhost:3030`。

### ckm:slides

- **路径**：`/root/.codex/skills/ui-ux-pro-max/slides/SKILL.md`
- **用途**：创建战略型 HTML 演示文稿，结合 Chart.js、设计 token、响应式布局、文案公式和情境化 slide 策略。
- **最佳使用场景**：营销演示、pitch deck、数据驱动型 slides、需要文案策略和图表支撑的 HTML 演示页面。
- **入口命令或触发**：入口名为 `ckm:slides`，参数提示为 `[topic] [slide-count]`；子命令 `create` 对应 `references/create.md`。

### ckm:design-system

- **路径**：`/root/.codex/skills/ui-ux-pro-max/design-system/SKILL.md`
- **用途**：提供 token 架构、组件规范、CSS 变量、间距/排版尺度，并支持品牌一致的 slide/presentation generation。
- **最佳使用场景**：需要建立或校验设计系统、生成品牌合规演示、把品牌色和组件规范系统化后再用于幻灯片或前端页面。
- **入口命令或触发**：入口名为 `ckm:design-system`，参数提示为 `[component or token]`；常用命令包括 `node scripts/generate-tokens.cjs --config tokens.json -o tokens.css`、`node scripts/validate-tokens.cjs --dir src/`，slide 相关脚本包括 `search-slides.py` 和 `slide-token-validator.py`。

### planning-map-generator-skill

- **路径**：`/root/.codex/skills/planning-map-generator-skill/SKILL.md`
- **用途**：为中国省、市、自治区、特别行政区生成可编辑 SVG 或 PPTX 规划地图，并处理 AMap/DataV 边界获取和本地地图生成器维护。
- **最佳使用场景**：演示或规划汇报中需要中国行政区地图、区域边界、可编辑 SVG/PPTX 地图输出，或需要调整 `map_generator.py`。
- **入口命令或触发**：触发场景包括中国行政区可编辑地图、AMap/DataV 边界检索、本地 planning-map-generator 工具改造；入口参考 `references/tool-map.md`，按要求输出到 `04_design-landscape\planning-map-generator-output` 下的日期子目录。

### yunnan-vector-map-tool

- **路径**：`/root/.codex/skills/yunnan-vector-map-tool/SKILL.md`
- **用途**：运行、检查或改造旧版云南/昆明矢量地图脚本、SVG 输出、PPTX 输出、区县 GeoJSON 和 WPS 转换宏。
- **最佳使用场景**：需要复用或再生成云南、昆明相关的旧版地图资产，检查 district geometry 输入，或处理 PowerPoint、WPS、Illustrator、Inkscape 兼容性。
- **入口命令或触发**：触发场景包括 legacy Yunnan vector-map workspace、云南/昆明 SVG/PPTX 变体、`yunnan_district.json`、WPS macro；入口参考 `references/tool-map.md`，运行前需先检查目标脚本和本地数据依赖。

## 转换/云工具

### markitdown

- **路径**：`/root/.codex/skills/markitdown/SKILL.md`
- **用途**：把 PDF、Office 文档、图片、音频、网页内容和结构化数据转换为适合 LLM 处理的 Markdown；支持 PPTX、DOCX、XLSX、PDF、HTML、EPUB、CSV、JSON 等格式。
- **最佳使用场景**：从 PowerPoint 或其他文件中抽取文本和结构，用于分析、总结、RAG、再生成大纲，或作为 PPT 生成流程的输入预处理。
- **入口命令或触发**：触发场景包括文档转 Markdown、PDF/Office 提取文本、OCR、音频转录、网页和 YouTube transcript 提取；命令行入口为 `markitdown document.pdf -o output.md`，Python 入口为 `from markitdown import MarkItDown` 后调用 `md.convert("document.pdf")`。

## 缺失候选文件

固定候选清单中的文件本次均可读取，未发现缺失项。

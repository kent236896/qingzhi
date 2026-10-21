# 知识库 Wiki 与图谱

知识库详情页在 `wiki_enabled=true` 时提供三个标签：**文档 | Wiki | 图谱**。选中标签与页面 slug 写入 URL query（例如 `?tab=wiki&slug=entity/lo-ra`），刷新可还原。

## 分组

| 左栏分组 | `page_type` |
|---|---|
| 知识 | `entity`、`concept`、`category` |
| 摘要 | `summary`、`synthesis`、`comparison` |

数量来自 `GET .../wiki/stats` 的 `pages_by_type` 聚合。索引页固定入口，数据来自 `GET .../wiki/index`。

## Wiki 链接语法

正文为 Markdown。内部链接实测为：

- `[[slug|标题]]`，例如 `[[entity/lo-ra|LoRA]]`
- 兼容 `[[slug]]`

前端渲染为绿色虚线下划线；目标不在标题缓存中时显示灰色并提示「页面不存在」。点击后在 Wiki 标签内跳转。

## 反向链接标题

WeKnora 页面只返回 `in_links` slug 列表。本系统提供 `GET /api/v1/knowledge-bases/:id/wiki/titles`，从索引分页汇总 slug→title，前端一次拉取后 O(1) 显示反向链接标题。

## 读代理接口

均需 `knowledge:query`；知识库未启用 Wiki 时返回 400。本系统统一响应信封；上游 WeKnora 多为原始对象。

- `GET /api/v1/knowledge-bases/:id/wiki/status`
- `GET /api/v1/knowledge-bases/:id/wiki/stats`
- `GET /api/v1/knowledge-bases/:id/wiki/graph`
- `GET /api/v1/knowledge-bases/:id/wiki/folders?parent_id=&page_types=`
- `GET /api/v1/knowledge-bases/:id/wiki/pages?folder_id=&page_type=&query=&page=&page_size=&sort_by=&sort_order=`（`page_size`≤50；`sort_by` 仅 `title` / `updated_at` / `created_at`）
- `GET /api/v1/knowledge-bases/:id/wiki/pages/*slug`（拒绝 `..`、反斜杠、控制字符、超长）
- `GET /api/v1/knowledge-bases/:id/wiki/index?types=&limit=&cursor=`（`limit`≤200）
- `GET /api/v1/knowledge-bases/:id/wiki/search?q=&limit=`（POSIX 正则转义，`q`≤100，`limit`≤20）
- `GET /api/v1/knowledge-bases/:id/wiki/titles`
- `GET /api/v1/knowledge-bases/:id/wiki/revisions/*slug`（只读预留，编辑能力后续交付）

写接口（新建/编辑页面与目录、移动、回滚）尚未开放，按第二/三阶段再做。

## 安全与风险

- 正文经 `marked` 转 HTML 后由 DOMPurify 清洗；外链仅保留 http/https，并加 `rel="noopener noreferrer"`。
- 当前实测样例正文**没有图片或内部文件引用**。若后续出现 WeKnora 文件地址，浏览器无法直连，需要另做带鉴权的文件代理后再改写图片 URL；本期未做。

## 图谱（对齐 WeKnora Wiki 图谱交互）

画布铺满标签页剩余区域；搜索 / 图例 / 适应屏幕 / 隐藏箭头为画布内浮层。`ready` 状态以标签栏旁「已完成 · N 个页面」表示，不再占整行横幅；`generating` / `empty` / `timeout` 仍用横幅。

| 能力 | 行为 |
|---|---|
| 选中 | 蓝色外环（`--qz-wiki-selection`）；相连边变蓝并略增粗；其余淡化；点空白取消 |
| 虚线环 | 仅 `ego` 且非中心、且 `link_count` 大于图中可见邻居时显示（表示还可加载） |
| 展开邻居 | `mode=ego&center=&depth=1`，**替换**画布为子图；双击同此 |
| 叠加邻居 | ego 下合并进当前图（上限 1500）；overview 下先 pivot 到 ego；Shift+单击同此 |
| 返回概览 | 重新拉取 `overview` |
| 详情抽屉 | 右侧覆盖约 460px；复用 `WikiPageBody`（与 Wiki 浏览器同一套 Markdown + DOMPurify）；正文内链只在图内定位/选中 |
| 色板 | CSS 变量 `--qz-wiki-*`（摘要蓝 / 实体绿 / 概念橙 / 综合浅蓝 / 对比红 / 索引灰） |

邻居提示文案与判定见 `apps/admin-web/src/utils/wikiNeighborStatus.ts`（移植自 WeKnora，MIT）。

### 布局调参与截图

现网可用 Wiki 约 11～12 个节点。100 / 500 节点用前端合成数据肉眼调参（斥力、边长、碰撞含标签、孤立节点底排、布局后再 `fitView`）。

建议在图谱标签对三组规模各截一张（标签疏密、箭头、选中态）：

1. 现网 ~11 节点 overview  
2. 合成 ~100 节点（浏览器控制台可临时 `graph.nodes` 扩样后热替换，或后端 limit 拉满）  
3. 合成 ~500 节点（触顶提示「仅展示部分节点」应出现）

截图可放在 `docs/wiki-graph-screenshots/`（目录按需创建，勿提交密钥或真实客户正文）。

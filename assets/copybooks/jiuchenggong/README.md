# 九成宫欧体范字库

本目录只保存可追溯的碑帖 manifest、来源说明和人工校审结果。高清扫描图、归一化页图、单字裁切图和后续字体工程产物不进入 git，应放在本地数据盘、对象存储或受控内容仓库。

## 推荐目录

```text
assets/copybooks/jiuchenggong/
├── manifest.sample.json      # 样例 manifest，不作为已审字库发布
├── source-images/            # 原始扫描页图，本目录被 .gitignore 忽略
├── normalized/               # 扶正/去噪/统一尺寸后的页图，本目录被 .gitignore 忽略
└── crops/                    # 单字裁切图，本目录被 .gitignore 忽略
```

## 导入原则

1. 只使用明确公版或已授权的扫描来源。
2. 每个 manifest 必须记录 `source_url`、`license_status` 和 `attribution`。
3. 每个单字必须记录 `source_image` 和像素级 `crop_box`。
4. `review_status=published` 才会被 API 对外返回。
5. AI 补字、部件合成字、人工重绘字必须单独标注，不得混入原碑裁切字。

## 校验命令

```bash
cd services/calligraphy
go run ./cmd/calligraphy-glyph-manifest validate ../../assets/copybooks/jiuchenggong/manifest.sample.json
```

## 运行时加载

```bash
CALLIGRAPHY_GLYPH_MANIFEST_FILE=/data/nebula-calligraphy/copybooks/jiuchenggong/manifest.json \
PORT=8090 \
go run ./cmd/calligraphy
```

配置后，manifest 中已发布的真实裁切字会优先于内置常用字样本返回；未发布或受限字不会出现在搜索和详情接口中。

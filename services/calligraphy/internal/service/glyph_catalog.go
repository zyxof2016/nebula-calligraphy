package service

import (
	"strings"

	"github.com/nebula-platform/nebula/services/calligraphy/internal/model"
)

type GlyphSearchParams = model.GlyphSearchParams

type GlyphCatalog interface {
	Search(params GlyphSearchParams) []model.Glyph
	GetDetail(glyphID string) (model.GlyphDetail, bool)
	ListPresetGroups(style string) []model.GlyphPresetGroup
}

type InMemoryGlyphCatalog struct {
	glyphs []model.Glyph
	groups []glyphPresetSeed
}

type glyphPresetSeed struct {
	id          string
	title       string
	description string
	characters  string
}

func NewInMemoryGlyphCatalog() *InMemoryGlyphCatalog {
	glyphs := []model.Glyph{
		seedGlyph("ou-jiuchenggong-shan", "山", "ou", "jiuchenggong", "欧阳询", "licensed", "published"),
		seedGlyph("ou-jiuchenggong-shui", "水", "ou", "jiuchenggong", "欧阳询", "licensed", "published"),
		seedGlyph("yan-duobaota-shui", "水", "yan", "duobaota", "颜真卿", "licensed", "published"),
		seedGlyph("yan-duobaota-qing", "清", "yan", "duobaota", "颜真卿", "licensed", "published"),
		seedGlyph("restricted-test", "禁", "ou", "private-copybook", "测试", "restricted", "published"),
	}
	groups := commonGlyphGroups()
	seen := map[string]bool{}
	for _, glyph := range glyphs {
		seen[glyph.Style+"-"+glyph.Character] = true
	}
	for _, styleSeed := range []struct {
		style        string
		copybookID   string
		calligrapher string
	}{
		{style: "ou", copybookID: "common-practice-ou", calligrapher: "欧阳询"},
		{style: "yan", copybookID: "common-practice-yan", calligrapher: "颜真卿"},
		{style: "liu", copybookID: "common-practice-liu", calligrapher: "柳公权"},
		{style: "zhao", copybookID: "common-practice-zhao", calligrapher: "赵孟頫"},
		{style: "slender_gold", copybookID: "common-practice-slender-gold", calligrapher: "宋徽宗"},
	} {
		for _, group := range groups {
			for _, character := range splitUniqueCharacters(group.characters) {
				key := styleSeed.style + "-" + character
				if seen[key] {
					continue
				}
				seen[key] = true
				glyphs = append(glyphs, seedGlyph(styleSeed.style+"-common-"+character, character, styleSeed.style, styleSeed.copybookID, styleSeed.calligrapher, "licensed", "published"))
			}
		}
	}
	return &InMemoryGlyphCatalog{glyphs: glyphs, groups: groups}
}

func (c *InMemoryGlyphCatalog) Search(params GlyphSearchParams) []model.Glyph {
	var matches []model.Glyph
	for _, glyph := range c.glyphs {
		if glyph.LicenseStatus == "restricted" || glyph.ReviewStatus != "published" {
			continue
		}
		if params.Character != "" && glyph.Character != params.Character {
			continue
		}
		if params.Style != "" && !strings.EqualFold(glyph.Style, params.Style) {
			continue
		}
		if params.CopybookID != "" && !strings.EqualFold(glyph.CopybookID, params.CopybookID) {
			continue
		}
		matches = append(matches, glyph)
	}
	return matches
}

func (c *InMemoryGlyphCatalog) GetDetail(glyphID string) (model.GlyphDetail, bool) {
	for _, glyph := range c.glyphs {
		if glyph.GlyphID != glyphID || glyph.LicenseStatus == "restricted" || glyph.ReviewStatus != "published" {
			continue
		}
		return model.GlyphDetail{
			Glyph: glyph,
			StructureNotes: []string{
				"先稳住中宫，再观察左右开合。",
				"临写时注意主笔方向和字势重心。",
			},
			BrushworkNotes: []string{
				"起笔宜沉着，行笔保持中锋感。",
				"收笔处放慢速度，避免轻浮散锋。",
			},
			PracticeTemplates: []model.PracticeTemplate{
				{TemplateType: "copy", GridType: "mi", Title: "米字格临摹", Description: "适合观察重心和斜向笔势。"},
				{TemplateType: "copy", GridType: "jiugong", Title: "九宫格结构", Description: "适合拆解上下左右比例。"},
				{TemplateType: "outline", GridType: "mi", Title: "双钩练习", Description: "适合描摹外轮廓和收放关系。"},
			},
		}, true
	}
	return model.GlyphDetail{}, false
}

func (c *InMemoryGlyphCatalog) ListPresetGroups(style string) []model.GlyphPresetGroup {
	if style == "" {
		style = "ou"
	}
	groups := make([]model.GlyphPresetGroup, 0, len(c.groups))
	for _, seed := range c.groups {
		glyphs := make([]model.Glyph, 0)
		for _, character := range splitUniqueCharacters(seed.characters) {
			for _, glyph := range c.glyphs {
				if glyph.Character == character && glyph.Style == style && glyph.LicenseStatus != "restricted" && glyph.ReviewStatus == "published" {
					glyphs = append(glyphs, glyph)
					break
				}
			}
		}
		groups = append(groups, model.GlyphPresetGroup{
			GroupID:     seed.id,
			Title:       seed.title,
			Description: seed.description,
			Glyphs:      glyphs,
		})
	}
	return groups
}

func seedGlyph(id, character, style, copybookID, calligrapher, licenseStatus, reviewStatus string) model.Glyph {
	return model.Glyph{
		GlyphID:       id,
		Character:     character,
		Style:         style,
		CopybookID:    copybookID,
		Calligrapher:  calligrapher,
		SourceImage:   "s3://nebula-calligraphy/seed/" + id + ".png",
		CropBox:       model.CropBox{X: 0, Y: 0, Width: 1, Height: 1, Unit: "ratio"},
		LicenseStatus: licenseStatus,
		ReviewStatus:  reviewStatus,
	}
}

func commonGlyphGroups() []glyphPresetSeed {
	return []glyphPresetSeed{
		{
			id:          "basic-strokes",
			title:       "基础笔画代表字",
			description: "覆盖横、竖、撇、捺、点、折、钩等入门观察。",
			characters:  "一二三十上土王工干于下不大天夫太犬人入八六小少水永火文方心必才寸中巾口日目田由甲申",
		},
		{
			id:          "structure",
			title:       "结构练习字",
			description: "适合观察独体、左右、上下、包围、穿插和避让。",
			characters:  "明林休体信仁何和知到新部都朝湖海清深语读诗书家安定空同国因园问间门闻风飞龙马鸟鱼",
		},
		{
			id:          "nature",
			title:       "自然意象常用字",
			description: "诗词与作品创作中常见的山水风物。",
			characters:  "山水云月日星风雨雪霜露花草木竹松梅兰菊石泉江河湖海春夏秋冬天地光影烟霞峰谷林溪桥舟",
		},
		{
			id:          "poetry",
			title:       "唐诗高频字",
			description: "覆盖日常集诗、临帖和短句创作的高频字。",
			characters:  "君我他谁何处来去归见闻知思忆梦心情意故乡长短高低远近前后东西南北古今夜朝白青红绿金玉",
		},
		{
			id:          "virtue",
			title:       "雅词与修身字",
			description: "适合日课、书斋小品和斗方创作。",
			characters:  "德道仁义礼智信诚敬静雅清和正气志学勤修养真善美乐寿福康宁平安吉祥如意自在无为",
		},
	}
}

func splitUniqueCharacters(characters string) []string {
	seen := map[string]bool{}
	out := make([]string, 0)
	for _, r := range characters {
		character := string(r)
		if seen[character] {
			continue
		}
		seen[character] = true
		out = append(out, character)
	}
	return out
}

class CropBox {
  const CropBox({
    required this.x,
    required this.y,
    required this.width,
    required this.height,
    required this.unit,
  });

  factory CropBox.fromJson(Map<String, dynamic> json) {
    return CropBox(
      x: _double(json['x']),
      y: _double(json['y']),
      width: _double(json['width']),
      height: _double(json['height']),
      unit: _string(json['unit'], fallback: 'px'),
    );
  }

  final double x;
  final double y;
  final double width;
  final double height;
  final String unit;
}

class RenderAsset {
  const RenderAsset({
    this.url = '',
    this.contentType = '',
    this.width = 0,
    this.height = 0,
    this.source = '',
  });

  factory RenderAsset.fromJson(Map<String, dynamic> json) {
    return RenderAsset(
      url: _string(json['url']),
      contentType: _string(json['content_type']),
      width: _int(json['width']),
      height: _int(json['height']),
      source: _string(json['source']),
    );
  }

  final String url;
  final String contentType;
  final int width;
  final int height;
  final String source;

  bool get isUsable => url.isNotEmpty;
}

class GlyphSummary {
  const GlyphSummary({
    required this.glyphId,
    required this.character,
    required this.style,
    required this.copybookId,
    required this.calligrapher,
    required this.sourceImage,
    required this.licenseStatus,
    required this.reviewStatus,
    this.renderAsset = const RenderAsset(),
    this.cropBox,
  });

  factory GlyphSummary.fromJson(Map<String, dynamic> json) {
    return GlyphSummary(
      glyphId: _string(json['glyph_id']),
      character: _string(json['character']),
      style: _string(json['style']),
      copybookId: _string(json['copybook_id']),
      calligrapher: _string(json['calligrapher']),
      sourceImage: _string(json['source_image']),
      cropBox: json['crop_box'] is Map<String, dynamic>
          ? CropBox.fromJson(json['crop_box'] as Map<String, dynamic>)
          : null,
      renderAsset: json['render_asset'] is Map<String, dynamic>
          ? RenderAsset.fromJson(json['render_asset'] as Map<String, dynamic>)
          : const RenderAsset(),
      licenseStatus: _string(json['license_status']),
      reviewStatus: _string(json['review_status']),
    );
  }

  final String glyphId;
  final String character;
  final String style;
  final String copybookId;
  final String calligrapher;
  final String sourceImage;
  final CropBox? cropBox;
  final RenderAsset renderAsset;
  final String licenseStatus;
  final String reviewStatus;
}

class PracticeTemplate {
  const PracticeTemplate({
    required this.templateType,
    required this.gridType,
    required this.title,
    required this.description,
  });

  factory PracticeTemplate.fromJson(Map<String, dynamic> json) {
    return PracticeTemplate(
      templateType: _string(json['template_type']),
      gridType: _string(json['grid_type']),
      title: _string(json['title']),
      description: _string(json['description']),
    );
  }

  final String templateType;
  final String gridType;
  final String title;
  final String description;
}

class GlyphDetail {
  const GlyphDetail({
    required this.glyph,
    required this.structureNotes,
    required this.brushworkNotes,
    required this.practiceTemplates,
  });

  factory GlyphDetail.fromJson(Map<String, dynamic> json) {
    return GlyphDetail(
      glyph: GlyphSummary.fromJson(json['glyph'] as Map<String, dynamic>),
      structureNotes: _stringList(json['structure_notes']),
      brushworkNotes: _stringList(json['brushwork_notes']),
      practiceTemplates: _objectList(
        json['practice_templates'],
        PracticeTemplate.fromJson,
      ),
    );
  }

  final GlyphSummary glyph;
  final List<String> structureNotes;
  final List<String> brushworkNotes;
  final List<PracticeTemplate> practiceTemplates;
}

class GlyphPresetGroup {
  const GlyphPresetGroup({
    required this.groupId,
    required this.title,
    required this.description,
    required this.glyphs,
  });

  factory GlyphPresetGroup.fromJson(Map<String, dynamic> json) {
    return GlyphPresetGroup(
      groupId: _string(json['group_id']),
      title: _string(json['title']),
      description: _string(json['description']),
      glyphs: _objectList(json['glyphs'], GlyphSummary.fromJson),
    );
  }

  final String groupId;
  final String title;
  final String description;
  final List<GlyphSummary> glyphs;
}

class PaperSpec {
  const PaperSpec({
    required this.format,
    required this.widthCm,
    required this.heightCm,
  });

  factory PaperSpec.fromJson(Map<String, dynamic> json) {
    return PaperSpec(
      format: _string(json['format']),
      widthCm: _double(json['width_cm']),
      heightCm: _double(json['height_cm']),
    );
  }

  Map<String, dynamic> toJson() => {
    'format': format,
    'width_cm': widthCm,
    'height_cm': heightCm,
  };

  final String format;
  final double widthCm;
  final double heightCm;
}

class LayoutRequest {
  const LayoutRequest({
    required this.text,
    required this.style,
    required this.copybookId,
    required this.paper,
    this.direction = 'vertical_rtl',
    this.marginCm = 3,
    this.signature = '',
    this.sealCount = 1,
  });

  Map<String, dynamic> toJson() => {
    'text': text,
    'style': style,
    'copybook_id': copybookId,
    'paper': paper.toJson(),
    'direction': direction,
    'margin_cm': marginCm,
    'signature': {'text': signature},
    'seal_count': sealCount,
  };

  final String text;
  final String style;
  final String copybookId;
  final PaperSpec paper;
  final String direction;
  final double marginCm;
  final String signature;
  final int sealCount;
}

class GlyphSlot {
  const GlyphSlot({
    required this.index,
    required this.character,
    required this.column,
    required this.row,
    required this.xCm,
    required this.yCm,
    required this.sizeCm,
    this.renderAsset = const RenderAsset(),
  });

  factory GlyphSlot.fromJson(Map<String, dynamic> json) {
    return GlyphSlot(
      index: _int(json['index']),
      character: _string(json['character']),
      column: _int(json['column']),
      row: _int(json['row']),
      xCm: _double(json['x_cm']),
      yCm: _double(json['y_cm']),
      sizeCm: _double(json['size_cm']),
      renderAsset: json['render_asset'] is Map<String, dynamic>
          ? RenderAsset.fromJson(json['render_asset'] as Map<String, dynamic>)
          : const RenderAsset(),
    );
  }

  final int index;
  final String character;
  final int column;
  final int row;
  final double xCm;
  final double yCm;
  final double sizeCm;
  final RenderAsset renderAsset;
}

class TextSlot {
  const TextSlot({
    required this.index,
    required this.text,
    required this.xCm,
    required this.yCm,
    required this.sizeCm,
  });

  factory TextSlot.fromJson(Map<String, dynamic> json) {
    return TextSlot(
      index: _int(json['index']),
      text: _string(json['text']),
      xCm: _double(json['x_cm']),
      yCm: _double(json['y_cm']),
      sizeCm: _double(json['size_cm']),
    );
  }

  final int index;
  final String text;
  final double xCm;
  final double yCm;
  final double sizeCm;
}

class LayoutResult {
  const LayoutResult({
    required this.layoutId,
    required this.normalizedText,
    required this.characterCount,
    required this.style,
    required this.copybookId,
    required this.paper,
    required this.direction,
    required this.marginCm,
    required this.columns,
    required this.rows,
    required this.glyphSizeCm,
    required this.slots,
    required this.signatureSlots,
    required this.sealSlots,
  });

  factory LayoutResult.fromJson(Map<String, dynamic> json) {
    return LayoutResult(
      layoutId: _string(json['layout_id']),
      normalizedText: _string(json['normalized_text']),
      characterCount: _int(json['character_count']),
      style: _string(json['style']),
      copybookId: _string(json['copybook_id']),
      paper: PaperSpec.fromJson(json['paper'] as Map<String, dynamic>),
      direction: _string(json['direction']),
      marginCm: _double(json['margin_cm']),
      columns: _int(json['columns']),
      rows: _int(json['rows']),
      glyphSizeCm: _double(json['glyph_size_cm']),
      slots: _objectList(json['slots'], GlyphSlot.fromJson),
      signatureSlots: _objectList(json['signature_slots'], TextSlot.fromJson),
      sealSlots: _objectList(json['seal_slots'], TextSlot.fromJson),
    );
  }

  final String layoutId;
  final String normalizedText;
  final int characterCount;
  final String style;
  final String copybookId;
  final PaperSpec paper;
  final String direction;
  final double marginCm;
  final int columns;
  final int rows;
  final double glyphSizeCm;
  final List<GlyphSlot> slots;
  final List<TextSlot> signatureSlots;
  final List<TextSlot> sealSlots;
}

class CreateArtworkDraftRequest {
  const CreateArtworkDraftRequest({
    required this.ownerUserId,
    required this.layout,
    this.glyphOverrides = const {},
  });

  Map<String, dynamic> toJson() => {
    'owner_user_id': ownerUserId,
    'layout': layout.toJson(),
    'glyph_overrides': glyphOverrides,
  };

  final String ownerUserId;
  final LayoutRequest layout;
  final Map<String, String> glyphOverrides;
}

class ArtworkDraft {
  const ArtworkDraft({
    required this.artworkId,
    required this.ownerUserId,
    required this.text,
    required this.layout,
    required this.createdAt,
    required this.updatedAt,
    required this.exports,
  });

  factory ArtworkDraft.fromJson(Map<String, dynamic> json) {
    return ArtworkDraft(
      artworkId: _string(json['artwork_id']),
      ownerUserId: _string(json['owner_user_id']),
      text: _string(json['text']),
      layout: LayoutResult.fromJson(json['layout'] as Map<String, dynamic>),
      createdAt: _string(json['created_at']),
      updatedAt: _string(json['updated_at']),
      exports: _objectList(json['exports'], ExportRecord.fromJson),
    );
  }

  final String artworkId;
  final String ownerUserId;
  final String text;
  final LayoutResult layout;
  final String createdAt;
  final String updatedAt;
  final List<ExportRecord> exports;
}

class ExportRecord {
  const ExportRecord({
    required this.exportId,
    required this.artworkId,
    required this.format,
    required this.templateType,
    required this.contentType,
    required this.sha256,
    required this.byteSize,
    required this.createdAt,
    this.storageKey = '',
    this.inlineContent = '',
    this.inlineEncoding = '',
  });

  factory ExportRecord.fromJson(Map<String, dynamic> json) {
    return ExportRecord(
      exportId: _string(json['export_id']),
      artworkId: _string(json['artwork_id']),
      format: _string(json['format']),
      templateType: _string(json['template_type']),
      contentType: _string(json['content_type']),
      storageKey: _string(json['storage_key']),
      sha256: _string(json['sha256']),
      byteSize: _int(json['byte_size']),
      inlineContent: _string(json['inline_content']),
      inlineEncoding: _string(json['inline_encoding']),
      createdAt: _string(json['created_at']),
    );
  }

  final String exportId;
  final String artworkId;
  final String format;
  final String templateType;
  final String contentType;
  final String storageKey;
  final String sha256;
  final int byteSize;
  final String inlineContent;
  final String inlineEncoding;
  final String createdAt;
}

class FavoriteGlyph {
  const FavoriteGlyph({
    required this.ownerUserId,
    required this.glyphId,
    required this.character,
    required this.style,
    required this.copybookId,
    required this.createdAt,
  });

  factory FavoriteGlyph.fromJson(Map<String, dynamic> json) {
    return FavoriteGlyph(
      ownerUserId: _string(json['owner_user_id']),
      glyphId: _string(json['glyph_id']),
      character: _string(json['character']),
      style: _string(json['style']),
      copybookId: _string(json['copybook_id']),
      createdAt: _string(json['created_at']),
    );
  }

  final String ownerUserId;
  final String glyphId;
  final String character;
  final String style;
  final String copybookId;
  final String createdAt;
}

class PracticeRecord {
  const PracticeRecord({
    required this.practiceId,
    required this.ownerUserId,
    required this.glyphId,
    required this.character,
    required this.style,
    required this.templateType,
    required this.gridType,
    required this.createdAt,
  });

  factory PracticeRecord.fromJson(Map<String, dynamic> json) {
    return PracticeRecord(
      practiceId: _string(json['practice_id']),
      ownerUserId: _string(json['owner_user_id']),
      glyphId: _string(json['glyph_id']),
      character: _string(json['character']),
      style: _string(json['style']),
      templateType: _string(json['template_type']),
      gridType: _string(json['grid_type']),
      createdAt: _string(json['created_at']),
    );
  }

  final String practiceId;
  final String ownerUserId;
  final String glyphId;
  final String character;
  final String style;
  final String templateType;
  final String gridType;
  final String createdAt;
}

class LearningProfile {
  const LearningProfile({
    required this.ownerUserId,
    required this.favorites,
    required this.recentPractice,
    required this.dailyPlan,
    required this.dailySteps,
    required this.practiceCount,
    required this.todayPracticeCount,
    required this.favoriteCount,
    this.lastPracticedAt = '',
  });

  factory LearningProfile.fromJson(Map<String, dynamic> json) {
    return LearningProfile(
      ownerUserId: _string(json['owner_user_id']),
      favorites: _objectList(json['favorites'], FavoriteGlyph.fromJson),
      recentPractice: _objectList(
        json['recent_practice'],
        PracticeRecord.fromJson,
      ),
      dailyPlan: _objectList(json['daily_plan'], PracticeSuggestion.fromJson),
      dailySteps: _objectList(json['daily_steps'], DailyPracticeStep.fromJson),
      practiceCount: _int(json['practice_count']),
      todayPracticeCount: _int(json['today_practice_count']),
      favoriteCount: _int(json['favorite_count']),
      lastPracticedAt: _string(json['last_practiced_at']),
    );
  }

  final String ownerUserId;
  final List<FavoriteGlyph> favorites;
  final List<PracticeRecord> recentPractice;
  final List<PracticeSuggestion> dailyPlan;
  final List<DailyPracticeStep> dailySteps;
  final int practiceCount;
  final int todayPracticeCount;
  final int favoriteCount;
  final String lastPracticedAt;
}

class PracticeSuggestion {
  const PracticeSuggestion({
    required this.glyphId,
    required this.character,
    required this.style,
    required this.copybookId,
    required this.title,
    required this.reason,
  });

  factory PracticeSuggestion.fromJson(Map<String, dynamic> json) {
    return PracticeSuggestion(
      glyphId: _string(json['glyph_id']),
      character: _string(json['character']),
      style: _string(json['style']),
      copybookId: _string(json['copybook_id']),
      title: _string(json['title']),
      reason: _string(json['reason']),
    );
  }

  final String glyphId;
  final String character;
  final String style;
  final String copybookId;
  final String title;
  final String reason;
}

class DailyPracticeStep {
  const DailyPracticeStep({
    required this.stepId,
    required this.title,
    required this.description,
    required this.actionLabel,
    required this.targetGlyphId,
    required this.targetCharacter,
    required this.completed,
  });

  factory DailyPracticeStep.fromJson(Map<String, dynamic> json) {
    return DailyPracticeStep(
      stepId: _string(json['step_id']),
      title: _string(json['title']),
      description: _string(json['description']),
      actionLabel: _string(json['action_label']),
      targetGlyphId: _string(json['target_glyph_id']),
      targetCharacter: _string(json['target_character']),
      completed: json['completed'] == true,
    );
  }

  final String stepId;
  final String title;
  final String description;
  final String actionLabel;
  final String targetGlyphId;
  final String targetCharacter;
  final bool completed;
}

class User {
  const User({
    required this.userId,
    required this.username,
    required this.createdAt,
  });

  factory User.fromJson(Map<String, dynamic> json) {
    return User(
      userId: _string(json['user_id']),
      username: _string(json['username']),
      createdAt: _string(json['created_at']),
    );
  }

  Map<String, dynamic> toJson() => {
    'user_id': userId,
    'username': username,
    'created_at': createdAt,
  };

  final String userId;
  final String username;
  final String createdAt;
}

class AuthSession {
  const AuthSession({
    required this.token,
    required this.expiresAt,
    required this.user,
  });

  factory AuthSession.fromJson(Map<String, dynamic> json) {
    return AuthSession(
      token: _string(json['token']),
      expiresAt: _string(json['expires_at']),
      user: User.fromJson(json['user'] as Map<String, dynamic>),
    );
  }

  final String token;
  final String expiresAt;
  final User user;
}

class RuntimeConfig {
  const RuntimeConfig({
    required this.runtimeProfile,
    required this.authMode,
    required this.identityClientId,
    required this.identityTenant,
    required this.identityAuthorizationEndpoint,
    required this.identityTokenEndpoint,
  });

  factory RuntimeConfig.fromJson(Map<String, dynamic> json) {
    return RuntimeConfig(
      runtimeProfile: _string(json['runtime_profile'], fallback: 'trial'),
      authMode: _string(json['auth_mode'], fallback: 'local'),
      identityClientId: _string(json['identity_client_id']),
      identityTenant: _string(json['identity_tenant']),
      identityAuthorizationEndpoint: _string(
        json['identity_authorization_endpoint'],
      ),
      identityTokenEndpoint: _string(json['identity_token_endpoint']),
    );
  }

  final String runtimeProfile;
  final String authMode;
  final String identityClientId;
  final String identityTenant;
  final String identityAuthorizationEndpoint;
  final String identityTokenEndpoint;
}

String _string(Object? value, {String fallback = ''}) {
  if (value == null) {
    return fallback;
  }
  return value.toString();
}

int _int(Object? value) {
  if (value is int) {
    return value;
  }
  if (value is num) {
    return value.round();
  }
  return int.tryParse(value?.toString() ?? '') ?? 0;
}

double _double(Object? value) {
  if (value is double) {
    return value;
  }
  if (value is num) {
    return value.toDouble();
  }
  return double.tryParse(value?.toString() ?? '') ?? 0;
}

List<String> _stringList(Object? value) {
  if (value is! List) {
    return const [];
  }
  return value.map((item) => item.toString()).toList(growable: false);
}

List<T> _objectList<T>(
  Object? value,
  T Function(Map<String, dynamic> json) fromJson,
) {
  if (value is! List) {
    return const [];
  }
  return value
      .whereType<Map<String, dynamic>>()
      .map(fromJson)
      .toList(growable: false);
}

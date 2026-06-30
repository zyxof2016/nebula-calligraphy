import 'package:flutter/foundation.dart';

import 'calligraphy_api.dart';
import 'models.dart';
import 'session_store.dart';

const selectableStyleOptions = <String, String>{
  'ou': '欧体',
  'yan': '颜体',
  'liu': '柳体',
  'zhao': '赵体',
  'slender_gold': '瘦金体',
};

const styleOptions = <String, String>{
  'ou': '欧体',
  'regular_ou': '欧体',
  'yan': '颜体',
  'regular_yan': '颜体',
  'liu': '柳体',
  'regular_liu': '柳体',
  'zhao': '赵体',
  'regular_zhao': '赵体',
  'slender_gold': '瘦金体',
};

String styleDisplayName(String style) => styleOptions[style] ?? style;

const paperOptions = <PaperSpec>[
  PaperSpec(format: '四尺整张', widthCm: 69, heightCm: 138),
  PaperSpec(format: '四尺对开', widthCm: 34, heightCm: 138),
  PaperSpec(format: '斗方', widthCm: 69, heightCm: 68),
  PaperSpec(format: '三尺整张', widthCm: 55, heightCm: 100),
];

class CalligraphyController extends ChangeNotifier {
  CalligraphyController({
    required this.gateway,
    SessionStore? sessionStore,
    required this.apiBaseUrl,
  }) : _sessionStore = sessionStore ?? MemorySessionStore();

  final CalligraphyGateway gateway;
  final SessionStore _sessionStore;
  final String apiBaseUrl;

  bool busy = false;
  String? errorMessage;
  User? currentUser;
  List<GlyphSummary> glyphs = const [];
  List<GlyphPresetGroup> presetGroups = const [];
  GlyphDetail? selectedGlyph;
  LayoutResult? layoutPreview;
  LayoutRequest? _lastLayoutRequest;
  ArtworkDraft? currentDraft;
  List<ArtworkDraft> drafts = const [];
  ExportRecord? lastExport;
  LearningProfile? learningProfile;
  String? practiceFeedback;
  String selectedStyle = 'ou';
  PaperSpec selectedPaper = paperOptions[2];
  int _nextPracticeIndex = 0;

  bool get isAuthenticated => currentUser != null;

  Future<void> initialize() async {
    final session = await _sessionStore.load();
    if (session == null) {
      return;
    }
    currentUser = session.user;
    gateway.setBearerToken(session.token);
    notifyListeners();
    await _refreshAuthenticatedData();
  }

  Future<void> login({
    required String username,
    required String password,
  }) async {
    await _authenticate(
      () => gateway.login(username: username, password: password),
    );
  }

  Future<void> register({
    required String username,
    required String password,
  }) async {
    await _authenticate(
      () => gateway.register(username: username, password: password),
    );
  }

  Future<void> logout() async {
    await _sessionStore.clear();
    gateway.setBearerToken(null);
    currentUser = null;
    learningProfile = null;
    drafts = const [];
    notifyListeners();
  }

  Future<void> searchGlyphs(String character) async {
    await _run(() async {
      glyphs = await gateway.searchGlyphs(
        character: character.trim(),
        style: selectedStyle,
      );
      if (glyphs.isNotEmpty) {
        await selectGlyph(glyphs.first.glyphId, notify: false);
      }
    });
  }

  Future<void> changePracticeGlyph() async {
    final next = practiceQueue[_nextPracticeIndex % practiceQueue.length];
    _nextPracticeIndex += 1;
    await searchGlyphs(next);
  }

  Future<void> loadPresets() async {
    await _run(() async {
      presetGroups = await gateway.listGlyphPresets(style: selectedStyle);
    });
  }

  Future<void> selectGlyph(String glyphId, {bool notify = true}) async {
    selectedGlyph = await gateway.getGlyphDetail(glyphId);
    if (notify) {
      notifyListeners();
    }
  }

  Future<void> previewCreation({required String text}) async {
    await _run(() async {
      _lastLayoutRequest = LayoutRequest(
        text: text.trim(),
        style: selectedStyle,
        copybookId: _copybookForStyle(selectedStyle),
        paper: selectedPaper,
        signature: '六月 试书',
      );
      layoutPreview = await gateway.previewLayout(_lastLayoutRequest!);
    });
  }

  Future<void> saveCurrentDraft() async {
    final user = currentUser;
    final layoutRequest = _lastLayoutRequest;
    if (user == null || layoutRequest == null) {
      return;
    }
    await _run(() async {
      currentDraft = await gateway.createDraft(
        CreateArtworkDraftRequest(
          ownerUserId: user.userId,
          layout: layoutRequest,
        ),
      );
      drafts = await gateway.listDrafts();
    });
  }

  Future<void> exportCurrentDraft({
    String format = 'svg',
    String templateType = 'reference',
  }) async {
    final draft = currentDraft;
    if (draft == null) {
      return;
    }
    await _run(() async {
      lastExport = await gateway.exportDraft(
        artworkId: draft.artworkId,
        format: format,
        templateType: templateType,
      );
    });
  }

  Future<void> recordSelectedGlyphPractice() async {
    final user = currentUser;
    final glyph = selectedGlyph?.glyph;
    if (user == null || glyph == null) {
      return;
    }
    await _run(() async {
      await gateway.recordPractice(
        ownerUserId: user.userId,
        glyphId: glyph.glyphId,
        templateType: 'trace',
        gridType: 'mi_grid',
      );
      learningProfile = await gateway.getLearningProfile(user.userId);
      practiceFeedback = '已记录 ✓';
    });
  }

  Future<void> _authenticate(Future<AuthSession> Function() request) async {
    await _run(() async {
      final session = await request();
      currentUser = session.user;
      gateway.setBearerToken(session.token);
      await _sessionStore.save(
        StoredSession(
          apiBaseUrl: apiBaseUrl,
          token: session.token,
          user: session.user,
        ),
      );
      await _refreshAuthenticatedData(notify: false);
    });
  }

  Future<void> _refreshAuthenticatedData({bool notify = true}) async {
    final user = currentUser;
    if (user == null) {
      return;
    }
    try {
      learningProfile = await gateway.getLearningProfile(user.userId);
      drafts = await gateway.listDrafts();
      presetGroups = await gateway.listGlyphPresets(style: selectedStyle);
      await _loadDefaultPracticeGlyph();
    } catch (error) {
      errorMessage = friendlyErrorMessage(error);
    }
    if (notify) {
      notifyListeners();
    }
  }

  Future<void> _loadDefaultPracticeGlyph() async {
    if (selectedGlyph != null) {
      return;
    }
    glyphs = await gateway.searchGlyphs(character: '永', style: selectedStyle);
    if (glyphs.isEmpty) {
      return;
    }
    selectedGlyph = await gateway.getGlyphDetail(glyphs.first.glyphId);
  }

  Future<void> _run(Future<void> Function() action) async {
    busy = true;
    errorMessage = null;
    notifyListeners();
    try {
      await action();
    } catch (error) {
      errorMessage = friendlyErrorMessage(error);
    } finally {
      busy = false;
      notifyListeners();
    }
  }

  void setSelectedStyle(String style) {
    selectedStyle = style;
    notifyListeners();
  }

  void setSelectedPaper(PaperSpec paper) {
    selectedPaper = paper;
    notifyListeners();
  }
}

const practiceQueue = ['水', '山', '月', '人', '心', '中', '和', '永'];

String friendlyErrorMessage(Object error) {
  final text = error.toString();
  if (text.contains('SocketException') ||
      text.contains('ClientException') ||
      text.contains('XMLHttpRequest') ||
      text.contains('Failed host lookup')) {
    return '网络连接失败，请稍后重试';
  }
  if (text.contains('glyph not found') || text.contains('not_found')) {
    return '这个字暂时没有范字，试试“永、山、水”';
  }
  if (text.contains('ApiException')) {
    final message = text.split(':').last.trim();
    return message.isEmpty ? '请求失败，请稍后重试' : message;
  }
  if (text.contains('http_error')) {
    return '请求失败，请稍后重试';
  }
  return text.isEmpty ? '操作失败，请稍后重试' : text;
}

String _copybookForStyle(String style) {
  return switch (style) {
    'yan' => 'duobaota',
    'liu' => 'xuanmita',
    'zhao' => 'danbabei',
    'slender_gold' => 'qianziwen',
    _ => 'jiuchenggong',
  };
}

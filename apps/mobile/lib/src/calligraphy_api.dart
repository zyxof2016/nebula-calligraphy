import 'dart:convert';

import 'package:http/http.dart' as http;

import 'models.dart';

abstract class CalligraphyGateway {
  void setBearerToken(String? token);

  Future<RuntimeConfig> getRuntimeConfig();

  Future<AuthSession> login({
    required String username,
    required String password,
  });

  Future<AuthSession> register({
    required String username,
    required String password,
  });

  Future<void> logout();

  Future<List<GlyphSummary>> searchGlyphs({
    String? character,
    String? style,
    String? copybookId,
  });

  Future<List<GlyphPresetGroup>> listGlyphPresets({String? style});

  Future<GlyphDetail> getGlyphDetail(String glyphId);

  Future<LayoutResult> previewLayout(LayoutRequest request);

  Future<ArtworkDraft> createDraft(CreateArtworkDraftRequest request);

  Future<List<ArtworkDraft>> listDrafts();

  Future<ExportRecord> exportDraft({
    required String artworkId,
    required String format,
    required String templateType,
  });

  Future<LearningProfile> getLearningProfile(String ownerUserId);

  Future<PracticeRecord> recordPractice({
    required String ownerUserId,
    required String glyphId,
    required String templateType,
    required String gridType,
  });
}

class ApiException implements Exception {
  const ApiException(this.statusCode, this.code, this.message);

  final int statusCode;
  final String code;
  final String message;

  @override
  String toString() => '$code: $message';
}

class CalligraphyApi implements CalligraphyGateway {
  CalligraphyApi({required this.baseUrl, http.Client? client})
    : _client = client ?? http.Client();

  final Uri baseUrl;
  final http.Client _client;
  String? _bearerToken;
  RuntimeConfig? _runtimeConfig;

  @override
  void setBearerToken(String? token) {
    _bearerToken = token;
  }

  @override
  Future<RuntimeConfig> getRuntimeConfig() async {
    final config = RuntimeConfig.fromJson(
      await _requestJson('GET', '/api/v1/calligraphy/runtime-config'),
    );
    _runtimeConfig = config;
    return config;
  }

  @override
  Future<AuthSession> login({
    required String username,
    required String password,
  }) async {
    final config = _runtimeConfig ?? await getRuntimeConfig();
    if (config.authMode == 'nebula-direct') {
      return _loginWithIdentity(config, username: username, password: password);
    }
    if (config.authMode != 'local') {
      throw const ApiException(
        400,
        'interactive_login_required',
        '当前部署要求使用 OIDC 登录入口',
      );
    }
    final json = await _requestJson(
      'POST',
      '/api/v1/calligraphy/auth/login',
      body: {'username': username, 'password': password},
    );
    return AuthSession.fromJson(json);
  }

  Future<AuthSession> _loginWithIdentity(
    RuntimeConfig config, {
    required String username,
    required String password,
  }) async {
    if (config.identityLoginEndpoint.isEmpty) {
      throw const ApiException(
        503,
        'identity_not_configured',
        'Identity 登录端点未配置',
      );
    }
    final response = await _client.post(
      Uri.parse(config.identityLoginEndpoint),
      headers: const {
        'accept': 'application/json',
        'content-type': 'application/json',
      },
      body: jsonEncode({'username': username, 'password': password}),
    );
    final payload = _decodeObject(response.body);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw ApiException(
        response.statusCode,
        payload['code']?.toString() ?? 'identity_login_failed',
        payload['message']?.toString() ?? 'Identity 登录失败',
      );
    }
    final data = payload['data'];
    final nested = data is Map<String, dynamic>
        ? data
        : const <String, dynamic>{};
    final token =
        payload['access_token']?.toString() ??
        payload['token']?.toString() ??
        nested['access_token']?.toString() ??
        nested['token']?.toString() ??
        '';
    if (token.isEmpty) {
      throw const ApiException(
        502,
        'identity_token_missing',
        'Identity 未返回访问令牌',
      );
    }
    _bearerToken = token;
    final user = User.fromJson(
      await _requestJson('GET', '/api/v1/calligraphy/auth/me'),
    );
    return AuthSession(
      token: token,
      expiresAt: _jwtExpiresAt(token),
      user: user,
    );
  }

  @override
  Future<AuthSession> register({
    required String username,
    required String password,
  }) async {
    final json = await _requestJson(
      'POST',
      '/api/v1/calligraphy/auth/register',
      body: {'username': username, 'password': password},
    );
    return AuthSession.fromJson(json);
  }

  @override
  Future<void> logout() async {
    await _requestJson('POST', '/api/v1/calligraphy/auth/logout');
  }

  @override
  Future<List<GlyphSummary>> searchGlyphs({
    String? character,
    String? style,
    String? copybookId,
  }) async {
    final json = await _requestJson(
      'GET',
      '/api/v1/calligraphy/glyphs/search',
      query: {
        'character': character,
        'style': style,
        'copybook_id': copybookId,
      },
    );
    return _items(json, GlyphSummary.fromJson);
  }

  @override
  Future<List<GlyphPresetGroup>> listGlyphPresets({String? style}) async {
    final json = await _requestJson(
      'GET',
      '/api/v1/calligraphy/glyphs/presets',
      query: {'style': style},
    );
    return _items(json, GlyphPresetGroup.fromJson);
  }

  @override
  Future<GlyphDetail> getGlyphDetail(String glyphId) async {
    final json = await _requestJson(
      'GET',
      '/api/v1/calligraphy/glyphs/$glyphId',
    );
    return GlyphDetail.fromJson(json);
  }

  @override
  Future<LayoutResult> previewLayout(LayoutRequest request) async {
    final json = await _requestJson(
      'POST',
      '/api/v1/calligraphy/layouts/preview',
      body: request.toJson(),
    );
    return LayoutResult.fromJson(json);
  }

  @override
  Future<ArtworkDraft> createDraft(CreateArtworkDraftRequest request) async {
    final json = await _requestJson(
      'POST',
      '/api/v1/calligraphy/artworks/drafts',
      body: request.toJson(),
    );
    return ArtworkDraft.fromJson(json);
  }

  @override
  Future<List<ArtworkDraft>> listDrafts() async {
    final json = await _requestJson(
      'GET',
      '/api/v1/calligraphy/artworks/drafts',
    );
    return _items(json, ArtworkDraft.fromJson);
  }

  @override
  Future<ExportRecord> exportDraft({
    required String artworkId,
    required String format,
    required String templateType,
  }) async {
    final json = await _requestJson(
      'POST',
      '/api/v1/calligraphy/artworks/drafts/$artworkId/exports',
      body: {'format': format, 'template_type': templateType},
    );
    return ExportRecord.fromJson(json);
  }

  @override
  Future<LearningProfile> getLearningProfile(String ownerUserId) async {
    final json = await _requestJson(
      'GET',
      '/api/v1/calligraphy/users/$ownerUserId/learning',
    );
    return LearningProfile.fromJson(json);
  }

  @override
  Future<PracticeRecord> recordPractice({
    required String ownerUserId,
    required String glyphId,
    required String templateType,
    required String gridType,
  }) async {
    final json = await _requestJson(
      'POST',
      '/api/v1/calligraphy/users/$ownerUserId/practice',
      body: {
        'glyph_id': glyphId,
        'template_type': templateType,
        'grid_type': gridType,
      },
    );
    return PracticeRecord.fromJson(json);
  }

  Future<Map<String, dynamic>> _requestJson(
    String method,
    String path, {
    Map<String, Object?>? query,
    Map<String, Object?>? body,
  }) async {
    final uri = _uri(path, query);
    final headers = <String, String>{'accept': 'application/json'};
    if (body != null) {
      headers['content-type'] = 'application/json';
    }
    if (_bearerToken != null && _bearerToken!.isNotEmpty) {
      headers['authorization'] = 'Bearer $_bearerToken';
    }

    final response = switch (method) {
      'GET' => await _client.get(uri, headers: headers),
      'POST' => await _client.post(
        uri,
        headers: headers,
        body: jsonEncode(body),
      ),
      _ => throw ArgumentError.value(method, 'method'),
    };

    final decoded = _decodeObject(response.body);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw ApiException(
        response.statusCode,
        decoded['code']?.toString() ??
            decoded['error_code']?.toString() ??
            'http_error',
        decoded['message']?.toString() ?? '请求失败',
      );
    }
    return decoded;
  }

  Uri _uri(String path, Map<String, Object?>? query) {
    final filteredQuery = <String, String>{};
    query?.forEach((key, value) {
      final text = value?.toString() ?? '';
      if (text.isNotEmpty) {
        filteredQuery[key] = text;
      }
    });

    final basePath = baseUrl.path.endsWith('/')
        ? baseUrl.path.substring(0, baseUrl.path.length - 1)
        : baseUrl.path;
    return baseUrl.replace(
      path: '$basePath$path',
      queryParameters: filteredQuery.isEmpty ? null : filteredQuery,
    );
  }
}

Map<String, dynamic> _decodeObject(String body) {
  if (body.trim().isEmpty) {
    return <String, dynamic>{};
  }
  try {
    final decoded = jsonDecode(body);
    return decoded is Map<String, dynamic>
        ? decoded
        : <String, dynamic>{'message': body};
  } on FormatException {
    return <String, dynamic>{'message': body};
  }
}

String _jwtExpiresAt(String token) {
  try {
    final parts = token.split('.');
    if (parts.length != 3) {
      throw const FormatException('invalid JWT');
    }
    final claims =
        jsonDecode(utf8.decode(base64Url.decode(base64Url.normalize(parts[1]))))
            as Map<String, dynamic>;
    final exp = claims['exp'];
    final seconds = exp is num ? exp.toInt() : int.parse(exp.toString());
    return DateTime.fromMillisecondsSinceEpoch(
      seconds * 1000,
      isUtc: true,
    ).toIso8601String();
  } catch (_) {
    return DateTime.now()
        .toUtc()
        .add(const Duration(hours: 1))
        .toIso8601String();
  }
}

List<T> _items<T>(
  Map<String, dynamic> json,
  T Function(Map<String, dynamic>) fromJson,
) {
  final items = json['items'];
  if (items is! List) {
    return const [];
  }
  return items
      .whereType<Map<String, dynamic>>()
      .map(fromJson)
      .toList(growable: false);
}

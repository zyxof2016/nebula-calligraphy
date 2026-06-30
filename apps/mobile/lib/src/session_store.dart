import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import 'models.dart';

class StoredSession {
  const StoredSession({
    required this.apiBaseUrl,
    required this.token,
    required this.user,
  });

  factory StoredSession.fromJson(Map<String, dynamic> json) {
    return StoredSession(
      apiBaseUrl: json['api_base_url']?.toString() ?? '',
      token: json['token']?.toString() ?? '',
      user: User.fromJson(json['user'] as Map<String, dynamic>),
    );
  }

  Map<String, dynamic> toJson() => {
    'api_base_url': apiBaseUrl,
    'token': token,
    'user': user.toJson(),
  };

  final String apiBaseUrl;
  final String token;
  final User user;
}

abstract class SessionStore {
  Future<StoredSession?> load();

  Future<void> save(StoredSession session);

  Future<void> clear();
}

class MemorySessionStore implements SessionStore {
  StoredSession? _session;

  @override
  Future<StoredSession?> load() async => _session;

  @override
  Future<void> save(StoredSession session) async {
    _session = session;
  }

  @override
  Future<void> clear() async {
    _session = null;
  }
}

class SharedPreferencesSessionStore implements SessionStore {
  static const _key = 'nebula_calligraphy_session';

  @override
  Future<StoredSession?> load() async {
    final prefs = await SharedPreferences.getInstance();
    final raw = prefs.getString(_key);
    if (raw == null || raw.isEmpty) {
      return null;
    }
    return StoredSession.fromJson(jsonDecode(raw) as Map<String, dynamic>);
  }

  @override
  Future<void> save(StoredSession session) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_key, jsonEncode(session.toJson()));
  }

  @override
  Future<void> clear() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_key);
  }
}

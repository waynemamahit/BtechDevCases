import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

class ApiException implements Exception {
  ApiException(this.message);

  final String message;

  @override
  String toString() => message;
}

class TransferEntry {
  TransferEntry({
    required this.transferId,
    required this.direction,
    required this.counterparty,
    required this.amount,
    required this.notes,
  });

  factory TransferEntry.fromJson(Map<String, dynamic> json) {
    return TransferEntry(
      transferId: json['transferId'] as String? ?? '',
      direction: json['direction'] as String? ?? '',
      counterparty: json['counterparty'] as String? ?? '',
      amount: (json['amount'] as num).toInt(),
      notes: json['notes'] as String? ?? '',
    );
  }

  final String transferId;
  final String direction;
  final String counterparty;
  final int amount;
  final String notes;
}

class WalletSnapshot {
  WalletSnapshot({required this.balance, required this.transfers});

  factory WalletSnapshot.fromJson(Map<String, dynamic> json) {
    final rows = json['transfers'];
    final transfers = <TransferEntry>[];
    if (rows is List) {
      for (final row in rows) {
        if (row is Map<String, dynamic>) {
          transfers.add(TransferEntry.fromJson(row));
        }
      }
    }
    return WalletSnapshot(
      balance: (json['balance'] as num).toInt(),
      transfers: transfers,
    );
  }

  final int balance;
  final List<TransferEntry> transfers;
}

class WalletView {
  int? balance;
  List<TransferEntry> transfers = [];
  String? error;

  void applyLoad(WalletSnapshot snapshot) {
    balance = snapshot.balance;
    transfers = List<TransferEntry>.from(snapshot.transfers);
    error = null;
  }

  void applyFailure() {
    error = 'Request failed';
  }
}

class ApiClient {
  ApiClient({
    required this.baseUrl,
    http.Client? httpClient,
    this.timeout = const Duration(seconds: 12),
  }) : _http = httpClient ?? http.Client();

  final String baseUrl;
  final Duration timeout;
  final http.Client _http;

  void close() => _http.close();

  Future<void> register({
    required String email,
    required String password,
    required String confirmPassword,
  }) async {
    await _send(
      'POST',
      '/register',
      body: {
        'email': email,
        'password': password,
        'confirmPassword': confirmPassword,
      },
    );
  }

  Future<String> login({
    required String email,
    required String password,
  }) async {
    final response = await _send(
      'POST',
      '/login',
      body: {'email': email, 'password': password},
    );
    final decoded = jsonDecode(response.body);
    if (decoded is! Map<String, dynamic> || decoded['token'] is! String) {
      throw ApiException('Request failed');
    }
    final token = decoded['token'] as String;
    if (token.isEmpty) {
      throw ApiException('Request failed');
    }
    return token;
  }

  Future<WalletSnapshot> wallet(String token) async {
    final response = await _send('GET', '/wallet', token: token);
    final decoded = jsonDecode(response.body);
    if (decoded is! Map<String, dynamic>) {
      throw ApiException('Request failed');
    }
    return WalletSnapshot.fromJson(decoded);
  }

  Future<void> transfer({
    required String token,
    required String transferId,
    required String recipient,
    required int amount,
    required String notes,
  }) async {
    await _send(
      'POST',
      '/transfers',
      token: token,
      body: {
        'transferId': transferId,
        'recipient': recipient,
        'amount': amount,
        'notes': notes,
      },
    );
  }

  Future<http.Response> _send(
    String method,
    String path, {
    String? token,
    Map<String, dynamic>? body,
  }) async {
    final headers = <String, String>{
      'Content-Type': 'application/json',
      if (token != null) 'Authorization': 'Bearer $token',
    };
    try {
      final call = method == 'GET'
          ? _http.get(Uri.parse('$baseUrl$path'), headers: headers)
          : _http.post(
              Uri.parse('$baseUrl$path'),
              headers: headers,
              body: jsonEncode(body),
            );
      final response = await call.timeout(timeout);
      if (response.statusCode < 200 || response.statusCode >= 300) {
        throw ApiException(_message(response));
      }
      return response;
    } on ApiException {
      rethrow;
    } catch (_) {
      throw ApiException('Request failed');
    }
  }

  String _message(http.Response response) {
    if (response.statusCode >= 500) {
      return 'Request failed';
    }
    try {
      final decoded = jsonDecode(response.body);
      if (decoded is Map<String, dynamic>) {
        final error = decoded['error'];
        if (error is String && error.isNotEmpty) {
          return error;
        }
      }
    } catch (_) {
      return 'Request failed';
    }
    return 'Request failed';
  }
}

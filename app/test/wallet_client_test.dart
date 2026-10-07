import 'dart:async';
import 'dart:convert';

import 'package:auth_wallet/api_client.dart';
import 'package:auth_wallet/money.dart';
import 'package:auth_wallet/session.dart';
import 'package:auth_wallet/transfer_draft.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

void main() {
  test('opening balance is shown with two decimal places', () {
    expect(formatMinor(100000), '1000.00');
    expect(formatMinor(100), '1.00');
  });

  test('a failed transfer keeps the balance and the transfer id', () {
    final view = WalletView();
    view.applyLoad(WalletSnapshot(balance: 100000, transfers: []));
    final ids = ['first', 'second'];
    var n = 0;
    final draft = TransferDraft(
      recipient: 'bob@example.com',
      amountText: '25.00',
      notes: '',
      newId: () => ids[n++],
    );
    final storedId = draft.transferId;

    view.applyFailure();

    expect(view.error, 'Request failed');
    expect(view.balance, 100000);
    expect(formatMinor(view.balance!), '1000.00');
    expect(draft.transferId, storedId);

    draft.edit(amountText: '1.00');
    expect(draft.transferId, 'second');
    expect(draft.transferId, isNot(storedId));
  });

  test('a failed call does not move the activity time', () {
    final start = DateTime.utc(2026, 10, 7, 12);
    var current = start;
    final session = Session(now: () => current);
    session.signIn(token: 'token', email: 'ada@example.com', at: start);
    final stored = session.lastSuccess;

    current = start.add(const Duration(minutes: 10));
    session.completeRequest(succeeded: false);

    expect(session.lastSuccess, stored);
  });

  test('the HTTP client times out after 12 seconds', () {
    final client = ApiClient(baseUrl: 'http://10.0.2.2:8080');
    addTearDown(client.close);
    expect(client.timeout, const Duration(seconds: 12));
  });

  test('validation errors from the API are kept', () async {
    const cases = <(int, String, String)>[
      (400, '{"error":"email is required"}', 'email is required'),
      (400, '{"error":"password is required"}', 'password is required'),
      (
        400,
        '{"error":"confirmPassword is required"}',
        'confirmPassword is required',
      ),
      (
        400,
        '{"error":"email is not a valid address"}',
        'email is not a valid address',
      ),
      (
        400,
        '{"error":"password must be non-empty"}',
        'password must be non-empty',
      ),
      (
        400,
        '{"error":"confirmPassword does not match password"}',
        'confirmPassword does not match password',
      ),
      (
        409,
        '{"error":"email is already registered"}',
        'email is already registered',
      ),
      (401, '{"error":"invalid credentials"}', 'invalid credentials'),
      (400, '{"error":"cannot transfer to self"}', 'cannot transfer to self'),
      (
        400,
        '{"error":"amount must be a positive integer"}',
        'amount must be a positive integer',
      ),
      (
        400,
        '{"error":"notes must be at most 200 characters"}',
        'notes must be at most 200 characters',
      ),
      (
        400,
        '{"error":"transferId must be 1 to 64 characters"}',
        'transferId must be 1 to 64 characters',
      ),
      (400, '{"error":"recipient is required"}', 'recipient is required'),
      (404, '{"error":"recipient not found"}', 'recipient not found'),
      (409, '{"error":"insufficient funds"}', 'insufficient funds'),
      (
        409,
        '{"error":"transfer id reused with a different payload"}',
        'transfer id reused with a different payload',
      ),
      (
        400,
        '{"error":"amount must be an integer"}',
        'amount must be an integer',
      ),
      (
        400,
        '{"error":"transferId, recipient, and amount are required"}',
        'transferId, recipient, and amount are required',
      ),
      (400, '{"error":"invalid JSON"}', 'invalid JSON'),
      (504, '{"error":"timeout"}', 'Request failed'),
      (500, '{"error":"could not transfer"}', 'Request failed'),
      (400, 'not-json', 'Request failed'),
      (400, '{"error":""}', 'Request failed'),
      (400, '{}', 'Request failed'),
    ];

    for (final (status, body, message) in cases) {
      final client = ApiClient(
        baseUrl: 'http://example.test',
        httpClient: _FixedClient(status, body),
      );
      addTearDown(client.close);
      await expectLater(
        client.register(
          email: 'ada@example.com',
          password: 'pw',
          confirmPassword: 'pw',
        ),
        throwsA(
          isA<ApiException>().having((err) => err.message, 'message', message),
        ),
      );
    }
  });

  test('a dropped connection is Request failed', () async {
    final client = ApiClient(
      baseUrl: 'http://example.test',
      httpClient: _FixedClient(0, '', throwOnSend: true),
    );
    addTearDown(client.close);
    await expectLater(
      client.login(email: 'ada@example.com', password: 'pw'),
      throwsA(
        isA<ApiException>().having(
          (err) => err.message,
          'message',
          'Request failed',
        ),
      ),
    );
  });
}

class _FixedClient extends http.BaseClient {
  _FixedClient(this.status, this.body, {this.throwOnSend = false});

  final int status;
  final String body;
  final bool throwOnSend;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    if (throwOnSend) {
      throw http.ClientException('down', request.url);
    }
    return http.StreamedResponse(
      Stream<List<int>>.value(utf8.encode(body)),
      status,
      headers: const {'content-type': 'application/json'},
    );
  }
}

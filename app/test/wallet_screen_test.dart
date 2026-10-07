import 'dart:async';
import 'dart:convert';

import 'package:auth_wallet/api_client.dart';
import 'package:auth_wallet/error_alert.dart';
import 'package:auth_wallet/main.dart';
import 'package:auth_wallet/session.dart';
import 'package:auth_wallet/transfer_history.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

void main() {
  testWidgets('error alert uses the error container', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(body: ErrorAlert(message: 'cannot transfer to self')),
      ),
    );

    expect(find.text('cannot transfer to self'), findsOneWidget);
    expect(find.byIcon(Icons.error_outline), findsOneWidget);
    final context = tester.element(find.byType(ErrorAlert));
    final material = tester.widget<Material>(
      find.descendant(
        of: find.byType(ErrorAlert),
        matching: find.byType(Material),
      ),
    );
    expect(material.color, Theme.of(context).colorScheme.errorContainer);
  });

  testWidgets('history keeps each transfer and its notes together', (
    tester,
  ) async {
    final transfers = [
      TransferEntry(
        transferId: 'in',
        direction: 'received',
        counterparty: 'grey@gmail.com',
        amount: 100000,
        notes: 'Try send another user',
      ),
      TransferEntry(
        transferId: 'out',
        direction: 'sent',
        counterparty: 'grey@gmail.com',
        amount: 75000,
        notes: 'Send back',
      ),
    ];

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(body: TransactionHistory(transfers: transfers)),
      ),
    );

    expect(find.text('History'), findsOneWidget);
    expect(find.byType(TransactionTile), findsNWidgets(2));
    expect(find.text('Received 1000.00'), findsOneWidget);
    expect(find.text('Sent 750.00'), findsOneWidget);
    expect(
      find.descendant(
        of: find.byType(TransactionTile).at(0),
        matching: find.text('Try send another user'),
      ),
      findsOneWidget,
    );
    expect(
      find.descendant(
        of: find.byType(TransactionTile).at(1),
        matching: find.text('Send back'),
      ),
      findsOneWidget,
    );
    expect(find.text('grey@gmail.com'), findsNWidgets(2));
  });

  testWidgets('an empty history says there are no transfers', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(body: TransactionHistory(transfers: [])),
      ),
    );

    expect(find.text('No transfers yet'), findsOneWidget);
    expect(find.byType(TransactionTile), findsNothing);
  });

  testWidgets('the wallet screen shows history inside transaction tiles', (
    tester,
  ) async {
    final api = await _openWallet(tester);

    expect(
      find.text('Hello wayne.mamahit@gmail.com, welcome back'),
      findsOneWidget,
    );
    expect(find.text('1250.00'), findsOneWidget);
    expect(find.byType(TransactionTile), findsNWidgets(2));
    expect(find.text('Received 1000.00'), findsOneWidget);
    expect(find.text('Sent 750.00'), findsOneWidget);
    expect(find.text('Try send another user'), findsOneWidget);
    expect(find.text('Send back'), findsOneWidget);
    expect(api.transferCalls, 0);
  });

  testWidgets('local validation errors all show in the alert', (tester) async {
    final api = await _openWallet(tester);

    await tester.enterText(_field('Amount'), '0');
    await tester.enterText(_field('Notes'), 'n' * 201);
    await tester.tap(find.widgetWithText(FilledButton, 'Send'));
    await tester.pump();

    expect(find.byType(ErrorAlert), findsOneWidget);
    expect(find.textContaining('recipient is required'), findsOneWidget);
    expect(find.textContaining('amount must be positive'), findsOneWidget);
    expect(
      find.textContaining('notes must be at most 200 characters'),
      findsOneWidget,
    );
    expect(find.text('Retry'), findsNothing);
    expect(api.transferCalls, 0);
    expect(find.text('1250.00'), findsOneWidget);
  });

  testWidgets('a non-numeric amount shows the decimal validation error', (
    tester,
  ) async {
    await _openWallet(tester);
    await tester.enterText(_field('Recipient'), 'grey@gmail.com');
    await tester.enterText(_field('Amount'), '1.234');
    await tester.tap(find.widgetWithText(FilledButton, 'Send'));
    await tester.pump();

    expect(
      find.textContaining(
        'amount must be a positive number with at most two decimal places',
      ),
      findsOneWidget,
    );
    expect(find.byType(ErrorAlert), findsOneWidget);
  });

  testWidgets('a server validation error replaces Request failed', (
    tester,
  ) async {
    final api = await _openWallet(
      tester,
      transferStatus: 400,
      transferError: 'cannot transfer to self',
    );

    await tester.enterText(_field('Recipient'), 'wayne.mamahit@gmail.com');
    await tester.enterText(_field('Amount'), '750');
    await tester.enterText(_field('Notes'), 'Send to my self');
    await tester.tap(find.widgetWithText(FilledButton, 'Send'));
    await tester.pumpAndSettle();

    expect(find.text('cannot transfer to self'), findsOneWidget);
    expect(find.text('Request failed'), findsNothing);
    expect(find.byType(ErrorAlert), findsOneWidget);
    expect(find.text('Retry'), findsOneWidget);
    expect(find.text('1250.00'), findsOneWidget);
    expect(api.transferCalls, 1);
  });

  testWidgets('unknown recipient and insufficient funds stay visible', (
    tester,
  ) async {
    final api = await _openWallet(
      tester,
      transferStatus: 404,
      transferError: 'recipient not found',
    );
    await tester.enterText(_field('Recipient'), 'nobody@example.com');
    await tester.enterText(_field('Amount'), '1.00');
    await tester.tap(find.widgetWithText(FilledButton, 'Send'));
    await tester.pumpAndSettle();
    expect(find.text('recipient not found'), findsOneWidget);

    api.transferStatus = 409;
    api.transferError = 'insufficient funds';
    await tester.enterText(_field('Amount'), '9999.00');
    await tester.tap(find.widgetWithText(FilledButton, 'Send'));
    await tester.pumpAndSettle();
    expect(find.text('insufficient funds'), findsOneWidget);
    expect(find.text('1250.00'), findsOneWidget);
  });

  testWidgets('a timed out transfer shows Request failed', (tester) async {
    await _openWallet(tester, hangTransfer: true);
    await tester.enterText(_field('Recipient'), 'grey@gmail.com');
    await tester.enterText(_field('Amount'), '1.00');
    await tester.tap(find.widgetWithText(FilledButton, 'Send'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 40));
    await tester.pump();

    expect(find.text('Request failed'), findsOneWidget);
    expect(find.byType(ErrorAlert), findsOneWidget);
    expect(find.text('Retry'), findsOneWidget);
    expect(find.text('1250.00'), findsOneWidget);
  });

  testWidgets('login and register show their validation errors', (
    tester,
  ) async {
    final api = _ScriptedApi();
    api.loginStatus = 401;
    api.loginError = 'invalid credentials';
    await _pumpApp(tester, api);

    await tester.enterText(_field('Email'), 'ada@example.com');
    await tester.enterText(_field('Password'), 'nope');
    await tester.tap(find.widgetWithText(FilledButton, 'Login'));
    await tester.pumpAndSettle();

    expect(find.text('invalid credentials'), findsOneWidget);
    expect(find.byType(ErrorAlert), findsOneWidget);

    api.registerStatus = 409;
    api.registerError = 'email is already registered';
    await tester.tap(find.text('Create an account'));
    await tester.pumpAndSettle();
    await tester.enterText(_field('Email'), 'ada@example.com');
    await tester.enterText(_field('Password'), 'secret');
    await tester.enterText(_field('Confirm password'), 'secret');
    await tester.tap(find.widgetWithText(FilledButton, 'Register'));
    await tester.pumpAndSettle();

    expect(find.text('email is already registered'), findsOneWidget);
    expect(find.byType(ErrorAlert), findsOneWidget);
  });
}

Finder _field(String label) {
  return find.byWidgetPredicate(
    (widget) => widget is TextField && widget.decoration?.labelText == label,
  );
}

Future<void> _pumpApp(WidgetTester tester, _ScriptedApi api) async {
  await tester.binding.setSurfaceSize(const Size(500, 1600));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    AuthApp(
      client: ApiClient(
        baseUrl: 'http://example.test',
        httpClient: api,
        timeout: const Duration(milliseconds: 30),
      ),
      session: Session(),
    ),
  );
}

Future<_ScriptedApi> _openWallet(
  WidgetTester tester, {
  int transferStatus = 200,
  String? transferError,
  bool hangTransfer = false,
}) async {
  final api = _ScriptedApi()
    ..transferStatus = transferStatus
    ..transferError = transferError
    ..hangTransfer = hangTransfer;
  await _pumpApp(tester, api);
  await tester.enterText(_field('Email'), 'wayne.mamahit@gmail.com');
  await tester.enterText(_field('Password'), 'secret');
  await tester.tap(find.widgetWithText(FilledButton, 'Login'));
  await tester.pumpAndSettle();
  return api;
}

class _ScriptedApi extends http.BaseClient {
  int loginStatus = 200;
  String? loginError;
  int registerStatus = 201;
  String? registerError;
  int transferStatus = 200;
  String? transferError;
  bool hangTransfer = false;
  int transferCalls = 0;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    if (request.url.path == '/transfers' && hangTransfer) {
      transferCalls++;
      return Completer<http.StreamedResponse>().future;
    }
    final response = _response(request);
    return http.StreamedResponse(
      Stream<List<int>>.value(utf8.encode(response.body)),
      response.statusCode,
      headers: const {'content-type': 'application/json'},
    );
  }

  http.Response _response(http.BaseRequest request) {
    switch (request.url.path) {
      case '/login':
        if (loginStatus >= 400) {
          return http.Response(jsonEncode({'error': loginError}), loginStatus);
        }
        return http.Response(jsonEncode({'token': 'token-1'}), 200);
      case '/register':
        if (registerStatus >= 400) {
          return http.Response(
            jsonEncode({'error': registerError}),
            registerStatus,
          );
        }
        return http.Response(
          jsonEncode({'id': '1', 'email': 'ada@example.com'}),
          201,
        );
      case '/wallet':
        return http.Response(
          jsonEncode({
            'balance': 125000,
            'transfers': [
              {
                'transferId': 'in',
                'direction': 'received',
                'counterparty': 'grey@gmail.com',
                'amount': 100000,
                'notes': 'Try send another user',
              },
              {
                'transferId': 'out',
                'direction': 'sent',
                'counterparty': 'grey@gmail.com',
                'amount': 75000,
                'notes': 'Send back',
              },
            ],
          }),
          200,
        );
      case '/transfers':
        transferCalls++;
        if (transferStatus >= 400) {
          return http.Response(
            jsonEncode({'error': transferError}),
            transferStatus,
          );
        }
        return http.Response(jsonEncode({'transferId': 'ok'}), 200);
      default:
        return http.Response(jsonEncode({'error': 'not found'}), 404);
    }
  }
}

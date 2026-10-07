import 'package:auth_wallet/welcome.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('welcome sentence', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(body: WelcomeBanner(email: 'ada@example.com')),
      ),
    );

    expect(find.text('Hello ada@example.com, welcome back'), findsOneWidget);
  });
}

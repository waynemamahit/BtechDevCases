import 'package:flutter/material.dart';

class WelcomeBanner extends StatelessWidget {
  const WelcomeBanner({super.key, required this.email});

  final String email;

  static String sentence(String email) => 'Hello $email, welcome back';

  @override
  Widget build(BuildContext context) {
    return Text(sentence(email));
  }
}

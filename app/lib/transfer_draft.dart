import 'dart:math';

import 'package:auth_wallet/money.dart';

class TransferDraft {
  TransferDraft({
    String? transferId,
    this.recipient = '',
    this.amountText = '',
    this.notes = '',
    String Function()? newId,
  }) : _newId = newId ?? newTransferId,
       transferId = transferId ?? (newId ?? newTransferId)();

  final String Function() _newId;
  String transferId;
  String recipient;
  String amountText;
  String notes;

  void edit({String? recipient, String? amountText, String? notes}) {
    final nextRecipient = recipient ?? this.recipient;
    final nextAmount = amountText ?? this.amountText;
    final nextNotes = notes ?? this.notes;
    final changed =
        nextRecipient != this.recipient ||
        nextAmount != this.amountText ||
        nextNotes != this.notes;
    if (changed) {
      transferId = _newId();
    }
    this.recipient = nextRecipient;
    this.amountText = nextAmount;
    this.notes = nextNotes;
  }

  String? validate() {
    final problems = <String>[];
    if (recipient.isEmpty) {
      problems.add('recipient is required');
    }
    try {
      parseAmount(amountText);
    } on MoneyParseException catch (err) {
      problems.add(err.message);
    }
    if (notes.runes.length > 200) {
      problems.add('notes must be at most 200 characters');
    }
    if (problems.isEmpty) {
      return null;
    }
    return problems.join('\n');
  }

  int get amount => parseAmount(amountText);
}

String newTransferId() {
  final rand = Random.secure();
  return List.generate(
    16,
    (_) => rand.nextInt(256).toRadixString(16).padLeft(2, '0'),
  ).join();
}

//connect to Mongo DB
conn = new Mongo();
db = conn.getDB("wallet");
//create and populate applications collection
db.getCollection('applications').insertMany([
  {
    "_id": "PAGOPA",
    "description": "",
    "status": "ENABLED",
    "creationDate": "2024-03-26T10:32:11.964744386Z",
    "updateDate": "2024-03-26T10:32:11.964746719Z",
    "_class": "it.pagopa.wallet.documents.applications.Application"
  }
]);

//create and populate payment-wallets collection
db.getCollection('payment-wallets').insertMany([
  {
    "_id": "b57a6eb5-1bd2-4e26-af17-074dd9fb341e",
    "userId": "00000000-0000-0000-0000-000000000000",
    "status": "VALIDATED",
    "paymentMethodId": "148ff003-46a6-4790-9376-b0e057352e45",
    "contractId": "contractId1",
    "validationOperationResult": "EXECUTED",
    "validationErrorCode": "000",
    "applications": [],
    "details": {
      "type": "CARDS",
      "bin": "12345678",
      "lastFourDigits": "1234",
      "expiryDate": "209912",
      "brand": "VISA",
      "paymentInstrumentGatewayId": "paymentInstrumentGatewayId1",
      "_class": "it.pagopa.wallet.documents.wallets.details.CardDetails"
    },
    "clients": {
      "IO": {
        "status": "ENABLED"
      }
    },
    "version": 4,
    "creationDate": "2024-06-17T14:58:40.428040266Z",
    "updateDate": "2024-06-17T14:58:52.796460285Z",
    "onboardingChannel": "IO",
    "_class": "it.pagopa.wallet.documents.wallets.Wallet"
  },
  {
    "_id": "763d8963-5338-4676-b978-52e2e72d0566",
    "userId": "00000000-0000-0000-0000-000000000000",
    "status": "VALIDATED",
    "paymentMethodId": "148ff003-46a6-4790-9376-b0e057352e45",
    "contractId": "contractId2",
    "validationOperationResult": "EXECUTED",
    "applications": [],
    "details": {
    "maskedEmail": "b***@icbpi.it",
    "pspId": "BCITITMM",
    "pspBusinessName": "Intesa Sanpaolo S.p.A",
    "_class": "it.pagopa.wallet.documents.wallets.details.PayPalDetails"
    },
    "clients": {
      "IO": {
        "status": "ENABLED"
      }
    },
    "version": 4,
    "creationDate": "2024-06-17T14:58:40.428040266Z",
    "updateDate": "2024-06-17T14:58:52.796460285Z",
    "onboardingChannel": "IO",
    "_class": "it.pagopa.wallet.documents.wallets.Wallet"
  },
  {
    "_id": "b57a6eb5-1bd2-4e26-af17-074dd9fb341f",
    "userId": "00000000-0000-0000-0000-000000000000",
    "status": "ERROR",
    "paymentMethodId": "148ff003-46a6-4790-9376-b0e057352e45",
    "contractId": "contractId3",
    "validationOperationResult": "FAILED",
    "validationErrorCode": "999",
    "applications": [],
    "details": {
      "type": "CARDS",
      "bin": "12345678",
      "lastFourDigits": "1234",
      "expiryDate": "209912",
      "brand": "VISA",
      "paymentInstrumentGatewayId": "paymentInstrumentGatewayId1",
      "_class": "it.pagopa.wallet.documents.wallets.details.CardDetails"
    },
    "clients": {
      "IO": {
        "status": "ENABLED"
      }
    },
    "version": 4,
    "creationDate": "2024-06-17T14:58:40.428040266Z",
    "updateDate": "2024-06-17T14:58:52.796460285Z",
    "onboardingChannel": "IO",
    "_class": "it.pagopa.wallet.documents.wallets.Wallet"
  },
  {
    "_id": "763d8963-5338-4676-b978-52e2e72d0567",
    "userId": "00000000-0000-0000-0000-000000000000",
    "status": "ERROR",
    "paymentMethodId": "148ff003-46a6-4790-9376-b0e057352e45",
    "contractId": "contractId4",
    "validationOperationResult": "FAILED",
    "applications": [],
    "details": {
    "maskedEmail": "b***@icbpi.it",
    "pspId": "BCITITMM",
    "pspBusinessName": "Intesa Sanpaolo S.p.A",
    "_class": "it.pagopa.wallet.documents.wallets.details.PayPalDetails"
    },
    "clients": {
      "IO": {
        "status": "ENABLED"
      }
    },
    "version": 4,
    "creationDate": "2024-06-17T14:58:40.428040266Z",
    "updateDate": "2024-06-17T14:58:52.796460285Z",
    "onboardingChannel": "IO",
    "_class": "it.pagopa.wallet.documents.wallets.Wallet"
  }
]);

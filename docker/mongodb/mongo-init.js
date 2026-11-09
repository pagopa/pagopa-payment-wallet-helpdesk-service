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

db.getCollection('wallet-contractId-deactivation-dlq').createIndex(
  { timestamp: 1, _id: 1 }
);
db.getCollection('payment-wallets-log-events').createIndex(
  { walletId: 1, timestamp: 1 }
);

db.getCollection('wallet-contractId-deactivation-dlq').insertMany([
  {
    "_id": "b57a6eb5-1bd2-4e26-af17-074dd9fb341e",
    "timestamp": "2026-08-11T13:24:47.978419012Z"
  },
  {
    "_id": "763d8963-5338-4676-b978-52e2e72d0566",
    "timestamp": "2026-08-11T13:24:47.978419012Z"
  },
  {
    "_id": "b57a6eb5-1bd2-4e26-af17-074dd9fb341f",
    "timestamp": "2026-08-11T13:24:47.978419013Z"
  }
]);

db.getCollection('payment-wallets-log-events').insertMany([
  {
    "_id": "00000000-0000-4000-8000-000000000001",
    "walletId": "b57a6eb5-1bd2-4e26-af17-074dd9fb341e",
    "contractId": "previous-contractId1",
    "timestamp": "2026-08-10T13:24:47.978419012Z",
    "errorType": "COMMUNICATION_ERROR",
    "attemptNumber": 1,
    "_class": "it.pagopa.wallet.audit.WalletContractIdDeactivationErrorEvent"
  },
  {
    "_id": "00000000-0000-4000-8000-000000000002",
    "walletId": "b57a6eb5-1bd2-4e26-af17-074dd9fb341e",
    "contractId": "contractId1",
    "timestamp": "2026-08-11T13:24:46.978419012Z",
    "errorType": "COMMUNICATION_ERROR",
    "httpErrorCode": 504,
    "attemptNumber": 1,
    "_class": "it.pagopa.wallet.audit.WalletContractIdDeactivationErrorEvent"
  },
  {
    "_id": "00000000-0000-4000-8000-000000000003",
    "walletId": "b57a6eb5-1bd2-4e26-af17-074dd9fb341e",
    "contractId": "contractId1",
    "timestamp": "2026-08-11T13:24:47.978419012Z",
    "errorType": "KO_RESPONSE_RECEIVED",
    "httpErrorCode": 500,
    "npgErrorCode": "PG_0017",
    "attemptNumber": 2,
    "_class": "it.pagopa.wallet.audit.WalletContractIdDeactivationErrorEvent"
  },
  {
    "_id": "00000000-0000-4000-8000-000000000004",
    "walletId": "763d8963-5338-4676-b978-52e2e72d0566",
    "contractId": "contractId2",
    "timestamp": "2026-08-11T13:24:47.978419012Z",
    "errorType": "COMMUNICATION_ERROR",
    "attemptNumber": 1,
    "_class": "it.pagopa.wallet.audit.WalletContractIdDeactivationErrorEvent"
  },
  {
    "_id": "00000000-0000-4000-8000-000000000005",
    "walletId": "b57a6eb5-1bd2-4e26-af17-074dd9fb341f",
    "contractId": "contractId3",
    "timestamp": "2026-08-11T13:24:47.978419013Z",
    "errorType": "KO_RESPONSE_RECEIVED",
    "httpErrorCode": 500,
    "npgErrorCode": "PG_0017",
    "attemptNumber": 1,
    "_class": "it.pagopa.wallet.audit.WalletContractIdDeactivationErrorEvent"
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
    "applications": [
      {
        "_id": "PAGOPA",
        "status": "ENABLED",
        "creationDate": "2099-01-01T00:00:00.000Z",
        "updateDate": "2099-01-01T00:00:00.000Z"
      }
    ],
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
    "applications": [
      {
        "_id": "PAGOPA",
        "status": "DISABLED",
        "creationDate": "2099-01-01T00:00:00.000Z",
        "updateDate": "2099-01-01T00:00:00.000Z"
      }
    ],
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

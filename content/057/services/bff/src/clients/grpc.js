import path from 'node:path';
import { fileURLToPath } from 'node:url';
import grpc from '@grpc/grpc-js';
import protoLoader from '@grpc/proto-loader';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const PROTO_PATH = path.resolve(__dirname, '../../proto/product.proto');

const pkgDef = protoLoader.loadSync(PROTO_PATH, {
  keepCase: false,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

const proto = grpc.loadPackageDefinition(pkgDef);
const ProductService = proto.product.v1.ProductService;

const target = process.env.PRODUCT_GRPC_ADDR || 'product:50051';
const client = new ProductService(target, grpc.credentials.createInsecure());

function promisify(method) {
  return (req) =>
    new Promise((resolve, reject) => {
      client[method](req, (err, resp) => (err ? reject(err) : resolve(resp)));
    });
}

export const productGrpc = {
  get: promisify('GetProduct'),
  list: promisify('ListProducts'),
  create: promisify('CreateProduct'),
  decrementStock: promisify('DecrementStock'),
};

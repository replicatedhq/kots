import gql from "graphql-tag";

export const uploadImageWatchBatch = gql`
  mutation uploadImageWatchBatch($imageList: String!) {
    uploadImageWatchBatch(imageList: $imageList)
  }
`;

package grpcbuild

import (
	"context"
	"fmt"
	"io"
	"time"

	log "github.com/sirupsen/logrus"

	pb "github.com/quay/quay-builder/buildman_pb"
	"github.com/quay/quay-builder/rpc"

	"google.golang.org/grpc"
)

type grpcClient struct {
	client           pb.BuildManagerClient
	currentPhase     rpc.Phase
	jobToken         string
	logStream        pb.BuildManager_LogMessageClient
	logSequenceNum   int
	phaseSequenceNum int
}

func NewClient(ctx context.Context, conn *grpc.ClientConn) (rpc.Client, error) {
	bmClient := pb.NewBuildManagerClient(conn)
	client := &grpcClient{
		client: bmClient,
	}

	if ok, err := client.Ping(); !ok {
		return nil, err
	}

	// Create log stream
	log.Infof("starting log stream to buildmanager")
	logStream, err := bmClient.LogMessage(ctx)
	if err != nil {
		return nil, err
	}

	client.logStream = logStream

	return client, nil
}

func (c *grpcClient) Ping() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := c.client.Ping(ctx, &pb.PingRequest{})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *grpcClient) RegisterBuildJob(registrationToken string) (*rpc.BuildArgs, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	buildpack, err := c.client.RegisterBuildJob(ctx, &pb.BuildJobArgs{RegisterJwt: registrationToken})
	if err != nil {
		return nil, err
	}

	c.jobToken = buildpack.GetJobJwt()

	buildArgs := &rpc.BuildArgs{
		Context:        buildpack.Context,
		DockerfilePath: buildpack.DockerfilePath,
		Repository:     buildpack.Repository,
		Registry:       buildpack.Registry,
		PullToken:      buildpack.PullToken,
		PushToken:      buildpack.PushToken,
		TagNames:       buildpack.TagNames,
		BaseImage: rpc.BuildArgsBaseImage{
			Username: buildpack.BaseImage.GetUsername(),
			Password: buildpack.BaseImage.GetPassword(),
		},
	}

	switch bp := buildpack.BuildPack.(type) {
	case *pb.BuildPack_PackageUrl:
		buildArgs.BuildPackage = bp.PackageUrl
	case *pb.BuildPack_GitPackage_:
		buildArgs.Git = &rpc.BuildArgsGit{
			URL:        bp.GitPackage.GetUrl(),
			SHA:        bp.GitPackage.GetSha(),
			PrivateKey: bp.GitPackage.GetPrivateKey(),
		}
	default:
		return nil, fmt.Errorf("Buildpack.Buildpack has unexpected type %T", bp)
	}

	return buildArgs, nil
}

func (c *grpcClient) SetPhase(phase rpc.Phase, pmd *rpc.PullMetadata) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c.currentPhase = phase
	c.phaseSequenceNum += 1

	statusData := &pb.SetPhaseRequest_PullMetadata{}
	if pmd != nil {
		statusData.RegistryUrl = pmd.RegistryURL
		statusData.BaseImage = pmd.BaseImage
		statusData.BaseImageTag = pmd.BaseImageTag
		statusData.PullUsername = pmd.PullUsername
	}

	phaseResponse, err := c.client.SetPhase(
		ctx,
		&pb.SetPhaseRequest{
			JobJwt:         c.jobToken,
			SequenceNumber: int32(c.phaseSequenceNum),
			Phase:          phaseEnum(phase),
			PullMetadata:   statusData,
		},
	)
	if err != nil {
		log.Errorf("failed to update phase: %v", err)
		return err
	}

	if !phaseResponse.Success {
		log.Errorf("build manager rejected phase transition: %v", err)
		return rpc.ErrClientRejectedPhaseTransition{}
	}

	if int(phaseResponse.SequenceNumber) != c.phaseSequenceNum {
		log.Errorf("build manager rejected phase transition (sequence out of order: %d vs %d)", phaseResponse.SequenceNumber, c.phaseSequenceNum)
		return rpc.ErrClientRejectedPhaseTransition{}
	}

	return nil
}

func (c *grpcClient) FindMostSimilarTag(tmd rpc.TagMetadata) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	baseImageName := tmd.BaseImage
	baseImageID := tmd.BaseImageID
	baseImageTag := tmd.BaseImageTag

	cacheTagResponse, err := c.client.DetermineCachedTag(
		ctx,
		&pb.CachedTagRequest{
			JobJwt:        c.jobToken,
			BaseImageName: baseImageName,
			BaseImageTag:  baseImageTag,
			BaseImageId:   baseImageID,
		},
	)
	if err != nil {
		return "", err
	}
	return cacheTagResponse.CachedTag, nil
}

func (c *grpcClient) PublishBuildLogEntry(entry string) error {
	c.logSequenceNum += 1
	err := c.logStream.Send(
		&pb.LogMessageRequest{
			JobJwt:         c.jobToken,
			SequenceNumber: int32(c.logSequenceNum),
			LogMessage:     entry,
		},
	)
	if err != nil {
		log.Warningf("failed to get log message: %s", err)
		c.logSequenceNum -= 1
		return err
	}

	logResp, err := c.logStream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		log.Warningf("failed to get log response: %s", err)
	}

	if !logResp.GetSuccess() {
		log.Warningf("buildmanager failed to log message: %d", c.logSequenceNum)
	}

	return nil
}

func (c *grpcClient) Heartbeat(ctx context.Context) {
	failedHeartbeatRetries := 3

	heartbeatStream, err := c.client.Heartbeat(ctx)
	if err != nil {
		log.Fatalf("failed to start heartbeat: %s", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if failedHeartbeatRetries == 0 {
				log.Fatalf("failed to update heartbeat too many times")
			}

			// Send heartbeat
			err := heartbeatStream.Send(&pb.HeartbeatRequest{JobJwt: c.jobToken})
			if err != nil {
				log.Warningf("failed to send heartbeat: %s", err)
				break
			}

			// Block until heartbeat response
			hearbeatResp, err := heartbeatStream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Warningf("failed to get heartbeat response: %s", err)
				break
			}

			if hearbeatResp.GetReply() {
				log.Infof("successfully sent heartbeat to BuildManager")
				failedHeartbeatRetries = 3
				break
			}

			// Retry if for some reason heartbeat was not updated
			if !hearbeatResp.GetReply() && failedHeartbeatRetries > 0 {
				log.Infof("heartbeat failed to update, retrying right away")
				failedHeartbeatRetries -= 1
				continue
			}
		}

		time.Sleep(2 * time.Second)
	}
}

func phaseEnum(phase rpc.Phase) pb.Phase {
	switch p := phase; p {
	case rpc.Waiting:
		return pb.Phase_WAITING
	case rpc.Unpacking:
		return pb.Phase_UNPACKING
	// TODO: Should CheckingCache and PrimingCache have separate phases.
	//       If so, the proto definition would need to be updated with the new phases.
	case rpc.Pulling, rpc.CheckingCache, rpc.PrimingCache:
		return pb.Phase_PULLING
	case rpc.Building:
		return pb.Phase_BUILDING
	case rpc.Pushing:
		return pb.Phase_PUSHING
	case rpc.Complete:
		return pb.Phase_COMPLETE
	case rpc.Error:
		return pb.Phase_ERROR
	default:
		return pb.Phase_ERROR
	}
}

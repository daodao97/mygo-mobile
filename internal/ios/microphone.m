//go:build ios && cgo

#include "native.h"

#if MYGO_IOS_MICROPHONE
#import <AVFoundation/AVFoundation.h>
#import "_cgo_export.h"

// One capture owns the shared audio session at a time. Every method runs on
// the main queue; the input tap runs on AVAudioEngine's render thread.
@interface MyGoMicrophoneCapture : NSObject
@property(nonatomic) uint64_t token;
@property(nonatomic, strong) AVAudioEngine *engine;
@property(nonatomic) BOOL tapped;
@property(nonatomic) BOOL active;
@property(nonatomic, strong) id interruption;
- (void)finish:(NSString *)error canceled:(BOOL)canceled;
@end

static MyGoMicrophoneCapture *microphoneCapture;

@implementation MyGoMicrophoneCapture
- (void)finish:(NSString *)error canceled:(BOOL)canceled {
    if (microphoneCapture != self) return;
    // The global is normally this object's last owner. Clearing it must not
    // destroy the receiver while cleanup or the Go callback still runs.
    __attribute__((objc_precise_lifetime)) MyGoMicrophoneCapture *keepAlive = self;
    uint64_t token = keepAlive.token;
    [self.engine stop];
    if (self.tapped) { [self.engine.inputNode removeTapOnBus:0]; self.tapped = NO; }
    if (self.active) {
        [[AVAudioSession sharedInstance] setActive:NO withOptions:AVAudioSessionSetActiveOptionNotifyOthersOnDeactivation error:nil];
        self.active = NO;
    }
    if (self.interruption) [[NSNotificationCenter defaultCenter] removeObserver:self.interruption];
    self.interruption = nil;
    microphoneCapture = nil;
    if (!canceled) goIOSMicrophoneAudio(token, NULL, 0, (char *)(error ? error.UTF8String : NULL), 1);
}
@end

static void microphoneOnMain(dispatch_block_t action) {
    if (NSThread.isMainThread) action(); else dispatch_sync(dispatch_get_main_queue(), action);
}

void mygo_ios_microphone_start(uint64_t token) {
    microphoneOnMain(^{
        if (microphoneCapture) { goIOSMicrophoneAudio(token, NULL, 0, "Microphone unavailable", 1); return; }
        MyGoMicrophoneCapture *capture = [MyGoMicrophoneCapture new];
        capture.token = token;
        capture.engine = [AVAudioEngine new];
        microphoneCapture = capture;
        AVAudioSession *audio = AVAudioSession.sharedInstance;
        NSError *error = nil;
        if (![audio setCategory:AVAudioSessionCategoryRecord mode:AVAudioSessionModeMeasurement options:AVAudioSessionCategoryOptionAllowBluetoothHFP error:&error]) {
            [capture finish:error.localizedDescription canceled:NO]; return;
        }
        // Audio input suppresses system haptics by default, including the
        // feedback an app gives that recording has started.
        [audio setAllowHapticsAndSystemSoundsDuringRecording:YES error:nil];
        if (![audio setActive:YES error:&error]) { [capture finish:error.localizedDescription canceled:NO]; return; }
        capture.active = YES;
        AVAudioInputNode *input = capture.engine.inputNode;
        AVAudioFormat *format = [input outputFormatForBus:0];
        if (format.sampleRate <= 0 || format.channelCount == 0) { [capture finish:@"Microphone unavailable" canceled:NO]; return; }
        AVAudioFormat *target = [[AVAudioFormat alloc] initWithCommonFormat:AVAudioPCMFormatInt16 sampleRate:16000 channels:1 interleaved:YES];
        AVAudioConverter *converter = [[AVAudioConverter alloc] initFromFormat:format toFormat:target];
        if (!converter) { [capture finish:@"Microphone unavailable" canceled:NO]; return; }
        __weak MyGoMicrophoneCapture *weak = capture;
        @try {
            [input installTapOnBus:0 bufferSize:1600 format:format block:^(AVAudioPCMBuffer *buffer, AVAudioTime *when) {
                AVAudioFrameCount capacity = (AVAudioFrameCount)ceil(buffer.frameLength * 16000.0 / format.sampleRate) + 32;
                AVAudioPCMBuffer *pcm = [[AVAudioPCMBuffer alloc] initWithPCMFormat:target frameCapacity:capacity];
                __block BOOL supplied = NO;
                NSError *failure = nil;
                [converter convertToBuffer:pcm error:&failure withInputFromBlock:^AVAudioBuffer *(AVAudioPacketCount count, AVAudioConverterInputStatus *status) {
                    if (supplied) { *status = AVAudioConverterInputStatus_NoDataNow; return nil; }
                    supplied = YES; *status = AVAudioConverterInputStatus_HaveData; return buffer;
                }];
                if (failure) { dispatch_async(dispatch_get_main_queue(), ^{ [weak finish:failure.localizedDescription canceled:NO]; }); return; }
                if (pcm.frameLength) goIOSMicrophoneAudio(token, pcm.int16ChannelData[0], (int)pcm.frameLength * 2, NULL, 0);
            }];
            capture.tapped = YES;
            [capture.engine prepare];
            if (![capture.engine startAndReturnError:&error]) { [capture finish:error.localizedDescription canceled:NO]; return; }
        } @catch (NSException *exception) { [capture finish:@"Microphone unavailable" canceled:NO]; return; }
        capture.interruption = [NSNotificationCenter.defaultCenter addObserverForName:AVAudioSessionInterruptionNotification object:audio queue:NSOperationQueue.mainQueue usingBlock:^(NSNotification *note) {
            if ([note.userInfo[AVAudioSessionInterruptionTypeKey] unsignedIntegerValue] == AVAudioSessionInterruptionTypeBegan) [weak finish:@"Recording interrupted" canceled:NO];
        }];
    });
}

void mygo_ios_microphone_stop(uint64_t token, int cancel) {
    microphoneOnMain(^{
        MyGoMicrophoneCapture *capture = microphoneCapture;
        if (capture && capture.token == token) [capture finish:nil canceled:cancel != 0];
    });
}
#else
#import "_cgo_export.h"
void mygo_ios_microphone_start(uint64_t token) { goIOSMicrophoneAudio(token, NULL, 0, "麦克风采集需要启用 microphone 能力", 1); }
void mygo_ios_microphone_stop(uint64_t token, int cancel) {}
#endif
